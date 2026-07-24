package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
)

type PostgresHandler struct {
	config PostgresConfig
	db     *sql.DB
}

var postgresDatabaseNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var errInvalidPostgresDatabase = errors.New("invalid or unreachable database")

const defaultSessionThresholdSecs int64 = 60

func parseSessionThreshold(param string) int64 {
	if param == "" {
		return defaultSessionThresholdSecs
	}
	value, err := strconv.ParseInt(param, 10, 64)
	if err != nil || value <= 0 {
		return defaultSessionThresholdSecs
	}
	return value
}

func buildPostgresDSN(config PostgresConfig, database string) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, config.Password),
		Host:     config.Host + ":" + config.Port,
		Path:     "/" + database,
		RawQuery: "sslmode=" + url.QueryEscape(config.SSLMode),
	}
	return u.String()
}

func openPostgresDB(config PostgresConfig, database string) (*sql.DB, error) {
	db, err := sql.Open("pgx", buildPostgresDSN(config, database))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	return db, nil
}

func NewPostgresHandler(config PostgresConfig) *PostgresHandler {
	db, err := openPostgresDB(config, config.Database)
	if err != nil {
		return &PostgresHandler{
			config: config,
			db:     nil,
		}
	}
	return &PostgresHandler{
		config: config,
		db:     db,
	}
}

func checkPostgresConnection(config PostgresConfig) error {
	db, err := openPostgresDB(config, config.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func (h *PostgresHandler) dbForDatabase(database string) (*sql.DB, func(), bool) {
	if database == "" || database == h.config.Database {
		return h.db, func() {}, true
	}
	if !postgresDatabaseNamePattern.MatchString(database) {
		return nil, func() {}, false
	}
	db, err := openPostgresDB(h.config, database)
	if err != nil {
		return nil, func() {}, false
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	return db, func() { db.Close() }, true
}

func splitSchemaTable(param string) (string, string) {
	if idx := strings.Index(param, "."); idx > 0 {
		return param[:idx], param[idx+1:]
	}
	return "public", param
}

func parsePgIntArray(s string) []int64 {
	trimmed := strings.Trim(s, "{}")
	if trimmed == "" {
		return []int64{}
	}
	parts := strings.Split(trimmed, ",")
	result := make([]int64, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			result = append(result, v)
		}
	}
	return result
}

func nullStrPtr(v sql.NullString) *string {
	if v.Valid {
		return &v.String
	}
	return nil
}

func nullTimeStr(v sql.NullTime) *string {
	if v.Valid {
		s := v.Time.UTC().Format(time.RFC3339)
		return &s
	}
	return nil
}

func (h *PostgresHandler) GetDatabases(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("PostgreSQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "PostgreSQL connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	recordPostgresOperation("get_databases", err)
	updatePostgresConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query PostgreSQL databases")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	databases := make([]string, 0)
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			log.Error().Err(err).Msg("Failed to scan database name")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
		databases = append(databases, dbName)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Error iterating database rows")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, DatabasesResponse{Databases: databases})
}

func (h *PostgresHandler) GetTables(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("PostgreSQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "PostgreSQL connection not available"})
		return
	}

	database := c.Param("database")

	db, closeDB, ok := h.dbForDatabase(database)
	defer closeDB()
	if !ok || db == nil {
		recordPostgresOperation("get_tables", errInvalidPostgresDatabase)
		log.Error().Str("database", database).Msg("Failed to open PostgreSQL database")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid or unreachable database: " + database})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT schemaname, tablename FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY schemaname, tablename")
	recordPostgresOperation("get_tables", err)
	updatePostgresConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Str("database", database).Msg("Failed to query PostgreSQL tables")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			log.Error().Err(err).Str("database", database).Msg("Failed to scan table name")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
		tables = append(tables, schemaName+"."+tableName)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Str("database", database).Msg("Error iterating table rows")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, TablesResponse{
		Database: database,
		Tables:   tables,
	})
}

func (h *PostgresHandler) DescribeTable(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("PostgreSQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "PostgreSQL connection not available"})
		return
	}

	database := c.Param("database")
	schema, table := splitSchemaTable(c.Param("table"))

	db, closeDB, ok := h.dbForDatabase(database)
	defer closeDB()
	if !ok || db == nil {
		recordPostgresOperation("describe_table", errInvalidPostgresDatabase)
		log.Error().Str("database", database).Msg("Failed to open PostgreSQL database")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid or unreachable database: " + database})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT column_name, data_type, is_nullable, column_default, character_maximum_length, ordinal_position, is_identity
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`
	rows, err := db.QueryContext(ctx, query, schema, table)
	recordPostgresOperation("describe_table", err)
	updatePostgresConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Str("database", database).Str("schema", schema).Str("table", table).Msg("Failed to describe PostgreSQL table")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	columns := make([]PostgresColumnInfo, 0)
	for rows.Next() {
		var name, dataType, nullable, isIdentity string
		var columnDefault sql.NullString
		var maxLength sql.NullInt64
		var position int64

		if err := rows.Scan(&name, &dataType, &nullable, &columnDefault, &maxLength, &position, &isIdentity); err != nil {
			log.Error().Err(err).Str("database", database).Str("schema", schema).Str("table", table).Msg("Failed to scan table column")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		column := PostgresColumnInfo{
			Name:       name,
			Type:       dataType,
			Nullable:   nullable,
			Position:   position,
			IsIdentity: isIdentity,
		}
		if columnDefault.Valid {
			column.Default = &columnDefault.String
		}
		if maxLength.Valid {
			column.MaxLength = &maxLength.Int64
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Str("database", database).Str("schema", schema).Str("table", table).Msg("Error iterating table column rows")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	response := PostgresTableDetailResponse{
		Database: database,
		Schema:   schema,
		Table:    table,
		Columns:  columns,
	}

	statsQuery := `SELECT
			coalesce(s.n_live_tup, 0), coalesce(s.n_dead_tup, 0), coalesce(s.seq_scan, 0), coalesce(s.idx_scan, 0),
			s.last_vacuum, s.last_autovacuum, s.last_analyze, s.last_autoanalyze,
			pg_total_relation_size(c.oid), pg_relation_size(c.oid), pg_indexes_size(c.oid), c.reltuples
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
		WHERE n.nspname = $1 AND c.relname = $2`

	var liveTuples, deadTuples, seqScans, indexScans, totalSize, tableSize, indexesSize int64
	var lastVacuum, lastAutovacuum, lastAnalyze, lastAutoanalyze sql.NullTime
	var estimatedRows float64

	err = db.QueryRowContext(ctx, statsQuery, schema, table).Scan(
		&liveTuples, &deadTuples, &seqScans, &indexScans,
		&lastVacuum, &lastAutovacuum, &lastAnalyze, &lastAutoanalyze,
		&totalSize, &tableSize, &indexesSize, &estimatedRows,
	)
	if err == nil {
		response.Stats = &PostgresTableStats{
			LiveTuples:       liveTuples,
			DeadTuples:       deadTuples,
			SeqScans:         seqScans,
			IndexScans:       indexScans,
			LastVacuum:       nullTimeStr(lastVacuum),
			LastAutovacuum:   nullTimeStr(lastAutovacuum),
			LastAnalyze:      nullTimeStr(lastAnalyze),
			LastAutoanalyze:  nullTimeStr(lastAutoanalyze),
			TotalSizeBytes:   totalSize,
			TableSizeBytes:   tableSize,
			IndexesSizeBytes: indexesSize,
			EstimatedRows:    estimatedRows,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *PostgresHandler) GetMetrics(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("PostgreSQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "PostgreSQL connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics := PostgresMetricsResponse{}

	metrics.Activity = h.getActivity(ctx)
	metrics.DatabaseStats = h.getDatabaseStats(ctx)
	metrics.Settings = h.getSettings(ctx)
	metrics.Replication = h.getReplicationInfo(ctx)
	metrics.Connections = h.getConnectionStats(ctx)
	metrics.Locks = h.getLockInfo(ctx)
	metrics.BgWriter = h.getBgWriterStats(ctx)
	metrics.Vacuum = h.getVacuumInfo(ctx)
	metrics.Sessions = h.getSessionIssues(ctx, parseSessionThreshold(c.Query("session-threshold")))

	recordPostgresOperation("get_metrics", nil)
	updatePostgresConnectionMetrics(h.db)
	c.JSON(http.StatusOK, metrics)
}

func (h *PostgresHandler) getActivity(ctx context.Context) []PgProcessInfo {
	query := `SELECT pid, usename, datname, application_name, client_addr::text, state,
			backend_start, xact_start, query_start, wait_event_type, wait_event, query
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid()`
	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query pg_stat_activity")
		return nil
	}
	defer rows.Close()

	activity := make([]PgProcessInfo, 0)
	for rows.Next() {
		var pid int64
		var user, database, appName, clientAddr, state, waitEventType, waitEvent, queryText sql.NullString
		var backendStart, xactStart, queryStart sql.NullTime

		err := rows.Scan(&pid, &user, &database, &appName, &clientAddr, &state,
			&backendStart, &xactStart, &queryStart, &waitEventType, &waitEvent, &queryText)
		if err != nil {
			continue
		}

		activity = append(activity, PgProcessInfo{
			PID:             pid,
			User:            nullStrPtr(user),
			Database:        nullStrPtr(database),
			ApplicationName: nullStrPtr(appName),
			ClientAddr:      nullStrPtr(clientAddr),
			State:           nullStrPtr(state),
			BackendStart:    nullTimeStr(backendStart),
			XactStart:       nullTimeStr(xactStart),
			QueryStart:      nullTimeStr(queryStart),
			WaitEventType:   nullStrPtr(waitEventType),
			WaitEvent:       nullStrPtr(waitEvent),
			Query:           nullStrPtr(queryText),
		})
	}
	return activity
}

func (h *PostgresHandler) getDatabaseStats(ctx context.Context) []PgDatabaseStats {
	query := `SELECT datname, xact_commit, xact_rollback, blks_read, blks_hit,
			tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted,
			conflicts, deadlocks, temp_files, temp_bytes
		FROM pg_stat_database
		WHERE datname IS NOT NULL`
	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query pg_stat_database")
		return nil
	}
	defer rows.Close()

	stats := make([]PgDatabaseStats, 0)
	for rows.Next() {
		var s PgDatabaseStats
		err := rows.Scan(&s.Database, &s.XactCommit, &s.XactRollback, &s.BlksRead, &s.BlksHit,
			&s.TupReturned, &s.TupFetched, &s.TupInserted, &s.TupUpdated, &s.TupDeleted,
			&s.Conflicts, &s.Deadlocks, &s.TempFiles, &s.TempBytes)
		if err != nil {
			continue
		}
		s.CacheHitRatio = computeCacheHitRatio(s.BlksHit, s.BlksRead)
		stats = append(stats, s)
	}
	return stats
}

func computeCacheHitRatio(hit, read int64) float64 {
	total := hit + read
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func (h *PostgresHandler) getSettings(ctx context.Context) map[string]string {
	query := `SELECT name, setting FROM pg_settings WHERE name IN (
		'max_connections', 'superuser_reserved_connections', 'shared_buffers', 'work_mem',
		'maintenance_work_mem', 'effective_cache_size', 'wal_level', 'max_wal_size',
		'checkpoint_timeout', 'autovacuum', 'autovacuum_max_workers', 'autovacuum_naptime',
		'max_worker_processes', 'max_parallel_workers', 'statement_timeout',
		'idle_in_transaction_session_timeout', 'server_version', 'data_directory' )`
	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query pg_settings")
		return nil
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var name, setting string
		if err := rows.Scan(&name, &setting); err == nil {
			settings[name] = setting
		}
	}
	return settings
}

func (h *PostgresHandler) getReplicationInfo(ctx context.Context) *PgReplicationInfo {
	info := &PgReplicationInfo{}

	if err := h.db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&info.IsInRecovery); err != nil {
		log.Error().Err(err).Msg("Failed to query pg_is_in_recovery")
		return nil
	}

	rows, err := h.db.QueryContext(ctx, `SELECT client_addr::text, application_name, state,
			sent_lsn::text, replay_lsn::text, write_lag::text, flush_lag::text, replay_lag::text
		FROM pg_stat_replication`)
	if err == nil {
		defer rows.Close()
		replicas := make([]PgReplicaInfo, 0)
		for rows.Next() {
			var clientAddr, sentLsn, replayLsn, writeLag, flushLag, replayLag sql.NullString
			var appName, state string
			err := rows.Scan(&clientAddr, &appName, &state, &sentLsn, &replayLsn, &writeLag, &flushLag, &replayLag)
			if err != nil {
				continue
			}
			replicas = append(replicas, PgReplicaInfo{
				ClientAddr:      nullStrPtr(clientAddr),
				ApplicationName: appName,
				State:           state,
				SentLsn:         nullStrPtr(sentLsn),
				ReplayLsn:       nullStrPtr(replayLsn),
				WriteLag:        nullStrPtr(writeLag),
				FlushLag:        nullStrPtr(flushLag),
				ReplayLag:       nullStrPtr(replayLag),
			})
		}
		info.Replicas = replicas
	}

	if info.IsInRecovery {
		var lastReplayLsn sql.NullString
		var replayLagSecs sql.NullFloat64
		err := h.db.QueryRowContext(ctx, `SELECT pg_last_wal_replay_lsn()::text,
			EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))::float8`).Scan(&lastReplayLsn, &replayLagSecs)
		if err == nil {
			info.LastReplayLsn = nullStrPtr(lastReplayLsn)
			if replayLagSecs.Valid {
				info.ReplayLagSecs = &replayLagSecs.Float64
			}
		}
	}

	return info
}

type pgConnGroup struct {
	App      string
	User     string
	Database string
	State    string
	Count    int64
}

func aggregatePostgresConnections(groups []pgConnGroup, maxConnections, reserved int64) *PostgresConnectionStats {
	stats := &PostgresConnectionStats{
		MaxConnections:       maxConnections,
		ReservedForSuperuser: reserved,
		ByApplication:        make([]ApplicationConnections, 0),
		ByUser:               make([]UserConnections, 0),
		ByDatabase:           make([]DatabaseConnections, 0),
		ByState:              make(map[string]int64),
	}

	appIdx := make(map[string]int)
	userCounts := make(map[string]int64)
	dbCounts := make(map[string]int64)

	for _, g := range groups {
		stats.Total += g.Count

		idx, ok := appIdx[g.App]
		if !ok {
			idx = len(stats.ByApplication)
			appIdx[g.App] = idx
			stats.ByApplication = append(stats.ByApplication, ApplicationConnections{
				ApplicationName: g.App,
				States:          make(map[string]int64),
			})
		}
		stats.ByApplication[idx].Count += g.Count
		if g.State != "" {
			stats.ByApplication[idx].States[g.State] += g.Count
		}

		if g.User != "" {
			userCounts[g.User] += g.Count
		}
		if g.Database != "" {
			dbCounts[g.Database] += g.Count
		}
		if g.State != "" {
			stats.ByState[g.State] += g.Count
		}
	}

	for user, count := range userCounts {
		stats.ByUser = append(stats.ByUser, UserConnections{User: user, Count: count})
	}
	for database, count := range dbCounts {
		stats.ByDatabase = append(stats.ByDatabase, DatabaseConnections{Database: database, Count: count})
	}

	sort.Slice(stats.ByApplication, func(i, j int) bool {
		if stats.ByApplication[i].Count != stats.ByApplication[j].Count {
			return stats.ByApplication[i].Count > stats.ByApplication[j].Count
		}
		return stats.ByApplication[i].ApplicationName < stats.ByApplication[j].ApplicationName
	})
	sort.Slice(stats.ByUser, func(i, j int) bool {
		if stats.ByUser[i].Count != stats.ByUser[j].Count {
			return stats.ByUser[i].Count > stats.ByUser[j].Count
		}
		return stats.ByUser[i].User < stats.ByUser[j].User
	})
	sort.Slice(stats.ByDatabase, func(i, j int) bool {
		if stats.ByDatabase[i].Count != stats.ByDatabase[j].Count {
			return stats.ByDatabase[i].Count > stats.ByDatabase[j].Count
		}
		return stats.ByDatabase[i].Database < stats.ByDatabase[j].Database
	})

	if maxConnections > 0 {
		stats.Available = maxConnections - reserved - stats.Total
		if stats.Available < 0 {
			stats.Available = 0
		}
	}

	return stats
}

func (h *PostgresHandler) getConnectionStats(ctx context.Context) *PostgresConnectionStats {
	var maxConnections, reserved int64
	err := h.db.QueryRowContext(ctx, `SELECT current_setting('max_connections')::bigint,
		current_setting('superuser_reserved_connections')::bigint`).Scan(&maxConnections, &reserved)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query connection settings")
	}

	rows, err := h.db.QueryContext(ctx, `SELECT coalesce(application_name, ''), coalesce(usename, ''),
			coalesce(datname, ''), coalesce(state, ''), count(*)
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid() AND backend_type = 'client backend'
		GROUP BY 1, 2, 3, 4`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query connection counts")
		return nil
	}
	defer rows.Close()

	groups := make([]pgConnGroup, 0)
	for rows.Next() {
		var g pgConnGroup
		if err := rows.Scan(&g.App, &g.User, &g.Database, &g.State, &g.Count); err != nil {
			continue
		}
		groups = append(groups, g)
	}

	return aggregatePostgresConnections(groups, maxConnections, reserved)
}

func (h *PostgresHandler) getLockInfo(ctx context.Context) *PgLockInfo {
	info := &PgLockInfo{}

	err := h.db.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE NOT granted) FROM pg_locks`).
		Scan(&info.TotalLocks, &info.UngrantedLocks)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query pg_locks")
		return nil
	}

	rows, err := h.db.QueryContext(ctx, `SELECT a.pid, a.query, a.usename, pg_blocking_pids(a.pid)::text
		FROM pg_stat_activity a
		WHERE cardinality(pg_blocking_pids(a.pid)) > 0`)
	if err == nil {
		defer rows.Close()
		blocked := make([]PgBlockedQuery, 0)
		for rows.Next() {
			var pid int64
			var queryText, user sql.NullString
			var blockingPids string
			if err := rows.Scan(&pid, &queryText, &user, &blockingPids); err != nil {
				continue
			}
			blocked = append(blocked, PgBlockedQuery{
				BlockedPID:   pid,
				BlockedQuery: nullStrPtr(queryText),
				BlockedUser:  nullStrPtr(user),
				BlockingPIDs: parsePgIntArray(blockingPids),
			})
		}
		if len(blocked) > 0 {
			info.BlockedQueries = blocked
		}
	}

	return info
}

func (h *PostgresHandler) getBgWriterStats(ctx context.Context) *PgBgWriterStats {
	stats := &PgBgWriterStats{}

	var checkpointsTimed, checkpointsReq, buffersCheckpoint, buffersClean, buffersBackend, buffersAlloc int64
	err := h.db.QueryRowContext(ctx, `SELECT checkpoints_timed, checkpoints_req, buffers_checkpoint,
			buffers_clean, buffers_backend, buffers_alloc
		FROM pg_stat_bgwriter`).
		Scan(&checkpointsTimed, &checkpointsReq, &buffersCheckpoint, &buffersClean, &buffersBackend, &buffersAlloc)
	if err == nil {
		stats.CheckpointsTimed = &checkpointsTimed
		stats.CheckpointsRequested = &checkpointsReq
		stats.BuffersCheckpoint = &buffersCheckpoint
		stats.BuffersClean = &buffersClean
		stats.BuffersBackend = &buffersBackend
		stats.BuffersAlloc = &buffersAlloc
		return stats
	}

	err = h.db.QueryRowContext(ctx, `SELECT num_timed, num_requested, buffers_written FROM pg_stat_checkpointer`).
		Scan(&checkpointsTimed, &checkpointsReq, &buffersCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query checkpoint statistics")
		return nil
	}
	stats.CheckpointsTimed = &checkpointsTimed
	stats.CheckpointsRequested = &checkpointsReq
	stats.BuffersCheckpoint = &buffersCheckpoint

	err = h.db.QueryRowContext(ctx, `SELECT buffers_clean, buffers_alloc FROM pg_stat_bgwriter`).
		Scan(&buffersClean, &buffersAlloc)
	if err == nil {
		stats.BuffersClean = &buffersClean
		stats.BuffersAlloc = &buffersAlloc
	}

	return stats
}

func (h *PostgresHandler) querySessionIssues(ctx context.Context, query string, thresholdSecs int64) []PgSessionIssue {
	rows, err := h.db.QueryContext(ctx, query, thresholdSecs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query session issues")
		return nil
	}
	defer rows.Close()

	issues := make([]PgSessionIssue, 0)
	for rows.Next() {
		var issue PgSessionIssue
		var user, database, appName, clientAddr, state, queryText sql.NullString
		if err := rows.Scan(&issue.PID, &user, &database, &appName, &clientAddr, &state, &queryText, &issue.DurationSecs); err != nil {
			continue
		}
		issue.User = nullStrPtr(user)
		issue.Database = nullStrPtr(database)
		issue.ApplicationName = nullStrPtr(appName)
		issue.ClientAddr = nullStrPtr(clientAddr)
		issue.State = nullStrPtr(state)
		issue.Query = nullStrPtr(queryText)
		issues = append(issues, issue)
	}
	return issues
}

func (h *PostgresHandler) getSessionIssues(ctx context.Context, thresholdSecs int64) *PgSessionIssues {
	longRunningQuery := `SELECT pid, usename, datname, application_name, client_addr::text, state, query,
			EXTRACT(EPOCH FROM (now() - query_start))::float8
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid() AND backend_type = 'client backend'
			AND state = 'active' AND query_start IS NOT NULL
			AND now() - query_start > make_interval(secs => $1)
		ORDER BY 8 DESC`

	idleInTransactionQuery := `SELECT pid, usename, datname, application_name, client_addr::text, state, query,
			EXTRACT(EPOCH FROM (now() - state_change))::float8
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid() AND backend_type = 'client backend'
			AND state IN ('idle in transaction', 'idle in transaction (aborted)')
			AND state_change IS NOT NULL
			AND now() - state_change > make_interval(secs => $1)
		ORDER BY 8 DESC`

	longRunning := h.querySessionIssues(ctx, longRunningQuery, thresholdSecs)
	idleInTransaction := h.querySessionIssues(ctx, idleInTransactionQuery, thresholdSecs)

	if longRunning == nil && idleInTransaction == nil {
		return nil
	}
	if longRunning == nil {
		longRunning = make([]PgSessionIssue, 0)
	}
	if idleInTransaction == nil {
		idleInTransaction = make([]PgSessionIssue, 0)
	}

	return &PgSessionIssues{
		ThresholdSecs:      thresholdSecs,
		LongRunningQueries: longRunning,
		IdleInTransaction:  idleInTransaction,
	}
}

func (h *PostgresHandler) getVacuumInfo(ctx context.Context) *PgVacuumInfo {
	info := &PgVacuumInfo{}

	rows, err := h.db.QueryContext(ctx, `SELECT schemaname, relname, n_dead_tup, n_live_tup, last_vacuum, last_autovacuum
		FROM pg_stat_user_tables
		ORDER BY n_dead_tup DESC
		LIMIT 10`)
	if err == nil {
		defer rows.Close()
		tables := make([]PgVacuumTableInfo, 0)
		for rows.Next() {
			var t PgVacuumTableInfo
			var lastVacuum, lastAutovacuum sql.NullTime
			if err := rows.Scan(&t.Schema, &t.Table, &t.DeadTuples, &t.LiveTuples, &lastVacuum, &lastAutovacuum); err != nil {
				continue
			}
			t.LastVacuum = nullTimeStr(lastVacuum)
			t.LastAutovacuum = nullTimeStr(lastAutovacuum)
			tables = append(tables, t)
		}
		info.TablesByDeadTuples = tables
	}

	var oldestXidAge sql.NullInt64
	err = h.db.QueryRowContext(ctx, `SELECT max(age(datfrozenxid)) FROM pg_database`).Scan(&oldestXidAge)
	if err == nil && oldestXidAge.Valid {
		info.OldestXidAge = &oldestXidAge.Int64
	}

	if len(info.TablesByDeadTuples) == 0 && info.OldestXidAge == nil {
		return nil
	}

	return info
}
