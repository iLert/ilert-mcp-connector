package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type ClickHouseHandler struct {
	config ClickHouseConfig
	conn   driver.Conn
}

func NewClickHouseHandler(config ClickHouseConfig) *ClickHouseHandler {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Username: config.User,
			Password: config.Password,
			Database: config.Database,
		},
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return &ClickHouseHandler{
			config: config,
			conn:   nil,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return &ClickHouseHandler{
			config: config,
			conn:   nil,
		}
	}

	return &ClickHouseHandler{
		config: config,
		conn:   conn,
	}
}

func checkClickHouseConnection(config ClickHouseConfig) error {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Username: config.User,
			Password: config.Password,
			Database: config.Database,
		},
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return conn.Ping(ctx)
}

func (h *ClickHouseHandler) GetDatabases(c *gin.Context) {
	if h.conn == nil {
		log.Error().Msg("ClickHouse connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "ClickHouse connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := h.conn.Query(ctx, "SHOW DATABASES")
	if err != nil {
		log.Error().Err(err).Msg("Failed to query ClickHouse databases")
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

	c.JSON(http.StatusOK, DatabasesResponse{Databases: databases})
}

func (h *ClickHouseHandler) GetTables(c *gin.Context) {
	if h.conn == nil {
		log.Error().Msg("ClickHouse connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "ClickHouse connection not available"})
		return
	}

	database := c.Param("database")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("SELECT name FROM system.tables WHERE database = '%s'", database)
	rows, err := h.conn.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Str("database", database).Msg("Failed to query ClickHouse tables")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Error().Err(err).Str("database", database).Msg("Failed to scan table name")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
		tables = append(tables, tableName)
	}

	c.JSON(http.StatusOK, TablesResponse{
		Database: database,
		Tables:   tables,
	})
}

func (h *ClickHouseHandler) DescribeTable(c *gin.Context) {
	if h.conn == nil {
		log.Error().Msg("ClickHouse connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "ClickHouse connection not available"})
		return
	}

	database := c.Param("database")
	table := c.Param("table")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get table columns
	query := fmt.Sprintf(`
		SELECT
			name, type, default_kind, default_expression, comment, codec_expression, ttl_expression
		FROM system.columns
		WHERE database = '%s' AND table = '%s'
		ORDER BY position
	`, database, table)

	rows, err := h.conn.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Str("database", database).Str("table", table).Msg("Failed to describe ClickHouse table")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	columns := make([]ClickHouseColumnInfo, 0)
	for rows.Next() {
		var col ClickHouseColumnInfo
		var defaultKind, defaultExpr, comment, codecExpr, ttlExpr *string

		err := rows.Scan(&col.Name, &col.Type, &defaultKind, &defaultExpr, &comment, &codecExpr, &ttlExpr)
		if err != nil {
			log.Error().Err(err).Str("database", database).Str("table", table).Msg("Failed to scan table column")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		if defaultKind != nil && *defaultKind != "" {
			col.DefaultKind = defaultKind
		}
		if defaultExpr != nil && *defaultExpr != "" {
			col.DefaultExpr = defaultExpr
		}
		if comment != nil && *comment != "" {
			col.Comment = comment
		}
		if codecExpr != nil && *codecExpr != "" {
			col.CodecExpr = codecExpr
		}
		if ttlExpr != nil && *ttlExpr != "" {
			col.TTLExpr = ttlExpr
		}

		columns = append(columns, col)
	}

	// Get table info
	infoQuery := fmt.Sprintf(`
		SELECT
			engine, total_rows, total_bytes, primary_key, sorting_key, partition_key
		FROM system.tables
		WHERE database = '%s' AND name = '%s'
	`, database, table)

	var engine, primaryKey, sortingKey, partitionKey string
	var totalRows, totalBytes uint64

	err = h.conn.QueryRow(ctx, infoQuery).Scan(&engine, &totalRows, &totalBytes, &primaryKey, &sortingKey, &partitionKey)
	if err != nil {
		log.Error().Err(err).Str("database", database).Str("table", table).Msg("Failed to get table info")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	response := map[string]interface{}{
		"database": database,
		"table":    table,
		"columns":  columns,
		"info": ClickHouseTableInfo{
			Database:     database,
			Table:        table,
			Engine:       engine,
			TotalRows:    totalRows,
			TotalBytes:   totalBytes,
			PrimaryKey:   primaryKey,
			SortingKey:   sortingKey,
			PartitionKey: partitionKey,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *ClickHouseHandler) GetMetrics(c *gin.Context) {
	if h.conn == nil {
		log.Error().Msg("ClickHouse connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "ClickHouse connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := ClickHouseMetrics{}

	// Get system.metrics
	metricsRows, err := h.conn.Query(ctx, "SELECT metric, value FROM system.metrics")
	if err == nil {
		defer metricsRows.Close()
		metricsMap := make(map[string]float64)
		for metricsRows.Next() {
			var metric string
			var value float64
			if err := metricsRows.Scan(&metric, &value); err == nil {
				metricsMap[metric] = value
			}
		}
		metrics.Metrics = metricsMap
	}

	// Get system.events
	eventsRows, err := h.conn.Query(ctx, "SELECT event, value FROM system.events")
	if err == nil {
		defer eventsRows.Close()
		eventsMap := make(map[string]uint64)
		for eventsRows.Next() {
			var event string
			var value uint64
			if err := eventsRows.Scan(&event, &value); err == nil {
				eventsMap[event] = value
			}
		}
		metrics.Events = eventsMap
	}

	// Get system.asynchronous_metrics
	asyncRows, err := h.conn.Query(ctx, "SELECT metric, value FROM system.asynchronous_metrics")
	if err == nil {
		defer asyncRows.Close()
		asyncMap := make(map[string]float64)
		for asyncRows.Next() {
			var metric string
			var value float64
			if err := asyncRows.Scan(&metric, &value); err == nil {
				asyncMap[metric] = value
			}
		}
		metrics.AsynchronousMetrics = asyncMap
	}

	// Get system.replicas (replication status)
	replicasRows, err := h.conn.Query(ctx, `
		SELECT
			database, table, is_leader, is_readonly, is_session_expired,
			future_parts, parts_to_check, zookeeper_path, replica_path,
			columns_version, queue_size, inserts_in_queue, merges_in_queue,
			log_max_index, log_pointer, total_replicas, active_replicas, lost_part_count
		FROM system.replicas
	`)
	if err == nil {
		defer replicasRows.Close()
		replicas := make([]ClickHouseReplica, 0)
		for replicasRows.Next() {
			var replica ClickHouseReplica
			err := replicasRows.Scan(
				&replica.Database, &replica.Table, &replica.IsLeader, &replica.IsReadonly,
				&replica.IsSessionExpired, &replica.FutureParts, &replica.PartsToCheck,
				&replica.ZookeeperPath, &replica.ReplicaPath, &replica.ColumnsVersion,
				&replica.QueueSize, &replica.InsertsInQueue, &replica.MergesInQueue,
				&replica.LogMaxIndex, &replica.LogPointer, &replica.TotalReplicas,
				&replica.ActiveReplicas, &replica.LostPartCount,
			)
			if err == nil {
				replicas = append(replicas, replica)
			}
		}
		if len(replicas) > 0 {
			metrics.Replicas = replicas
		}
	}

	// Get system.processes (running queries)
	processesRows, err := h.conn.Query(ctx, `
		SELECT
			query_id, user, address, elapsed, read_rows, read_bytes,
			total_rows_approx, written_rows, written_bytes, memory_usage, query
		FROM system.processes
	`)
	if err == nil {
		defer processesRows.Close()
		processes := make([]ClickHouseProcess, 0)
		for processesRows.Next() {
			var process ClickHouseProcess
			err := processesRows.Scan(
				&process.QueryID, &process.User, &process.Address, &process.Elapsed,
				&process.ReadRows, &process.ReadBytes, &process.TotalRows,
				&process.WrittenRows, &process.WrittenBytes, &process.MemoryUsage,
				&process.Query,
			)
			if err == nil {
				processes = append(processes, process)
			}
		}
		if len(processes) > 0 {
			metrics.Processes = processes
		}
	}

	// Get system.merges (merge operations)
	mergesRows, err := h.conn.Query(ctx, `
		SELECT
			database, table, elapsed, progress, num_parts_to_merge,
			rows_read, bytes_read_uncompressed, rows_written, bytes_written_uncompressed, memory_usage
		FROM system.merges
	`)
	if err == nil {
		defer mergesRows.Close()
		merges := make([]ClickHouseMerge, 0)
		for mergesRows.Next() {
			var merge ClickHouseMerge
			err := mergesRows.Scan(
				&merge.Database, &merge.Table, &merge.Elapsed, &merge.Progress,
				&merge.NumPartsToMerge, &merge.RowsRead, &merge.BytesRead,
				&merge.RowsWritten, &merge.BytesWritten, &merge.MemoryUsage,
			)
			if err == nil {
				merges = append(merges, merge)
			}
		}
		if len(merges) > 0 {
			metrics.Merges = merges
		}
	}

	// Get system.mutations (mutation operations)
	mutationsRows, err := h.conn.Query(ctx, `
		SELECT
			database, table, mutation_id, command, create_time, block_numbers,
			parts_to_do, is_done, latest_failed_part, latest_fail_time, latest_fail_reason
		FROM system.mutations
	`)
	if err == nil {
		defer mutationsRows.Close()
		mutations := make([]ClickHouseMutation, 0)
		for mutationsRows.Next() {
			var mutation ClickHouseMutation
			var latestFailedPart, latestFailTime, latestFailReason *string
			err := mutationsRows.Scan(
				&mutation.Database, &mutation.Table, &mutation.MutationID, &mutation.Command,
				&mutation.CreateTime, &mutation.BlockNumbers, &mutation.PartsToDo, &mutation.IsDone,
				&latestFailedPart, &latestFailTime, &latestFailReason,
			)
			if err == nil {
				if latestFailedPart != nil && *latestFailedPart != "" {
					mutation.LatestFailedPart = latestFailedPart
				}
				if latestFailTime != nil && *latestFailTime != "" {
					mutation.LatestFailTime = latestFailTime
				}
				if latestFailReason != nil && *latestFailReason != "" {
					mutation.LatestFailReason = latestFailReason
				}
				mutations = append(mutations, mutation)
			}
		}
		if len(mutations) > 0 {
			metrics.Mutations = mutations
		}
	}

	c.JSON(http.StatusOK, metrics)
}
