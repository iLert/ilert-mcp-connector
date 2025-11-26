package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

type MySQLHandler struct {
	config MySQLConfig
	db     *sql.DB
}

func NewMySQLHandler(config MySQLConfig) *MySQLHandler {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &MySQLHandler{
			config: config,
			db:     nil,
		}
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	return &MySQLHandler{
		config: config,
		db:     db,
	}
}

func checkMySQLConnection(config MySQLConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func (h *MySQLHandler) GetDatabases(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("MySQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "MySQL connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(ctx, "SHOW DATABASES")
	recordMySQLOperation("get_databases", err)
	updateMySQLConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query MySQL databases")
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

func (h *MySQLHandler) GetTables(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("MySQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "MySQL connection not available"})
		return
	}

	database := c.Param("database")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("SHOW TABLES FROM `%s`", database)
	rows, err := h.db.QueryContext(ctx, query)
	recordMySQLOperation("get_tables", err)
	updateMySQLConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Str("database", database).Msg("Failed to query MySQL tables")
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

func (h *MySQLHandler) DescribeTable(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("MySQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "MySQL connection not available"})
		return
	}

	database := c.Param("database")
	table := c.Param("table")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("DESCRIBE `%s`.`%s`", database, table)
	rows, err := h.db.QueryContext(ctx, query)
	recordMySQLOperation("describe_table", err)
	updateMySQLConnectionMetrics(h.db)
	if err != nil {
		log.Error().Err(err).Str("database", database).Str("table", table).Msg("Failed to describe MySQL table")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	columns := make([]ColumnInfo, 0)
	for rows.Next() {
		var field, fieldType, null, key, defaultValue sql.NullString
		var extra sql.NullString

		err := rows.Scan(&field, &fieldType, &null, &key, &defaultValue, &extra)
		if err != nil {
			log.Error().Err(err).Str("database", database).Str("table", table).Msg("Failed to scan table column")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		column := ColumnInfo{
			Field: field.String,
			Type:  fieldType.String,
			Null:  null.String,
			Key:   key.String,
		}

		if defaultValue.Valid {
			column.Default = &defaultValue.String
		}
		if extra.Valid {
			column.Extra = &extra.String
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Str("database", database).Str("table", table).Msg("Error iterating table column rows")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// Get table status
	var tableType, engine, rowFormat, tableRows, avgRowLength, dataLength, maxDataLength, indexLength, dataFree, autoIncrement, createTime, updateTime, checkTime, tableCollation, checksum, createOptions, tableComment sql.NullString

	statusQuery := fmt.Sprintf("SHOW TABLE STATUS FROM `%s` WHERE Name = '%s'", database, table)
	err = h.db.QueryRowContext(ctx, statusQuery).Scan(
		&tableType, &engine, &rowFormat, &tableRows, &avgRowLength, &dataLength,
		&maxDataLength, &indexLength, &dataFree, &autoIncrement, &createTime,
		&updateTime, &checkTime, &tableCollation, &checksum, &createOptions, &tableComment,
	)

	response := TableDetailResponse{
		Database: database,
		Table:    table,
		Columns:  columns,
	}

	if err == nil {
		response.Status = &TableStatus{
			Engine:         engine.String,
			RowFormat:      rowFormat.String,
			TableRows:      tableRows.String,
			AvgRowLength:   avgRowLength.String,
			DataLength:     dataLength.String,
			IndexLength:    indexLength.String,
			DataFree:       dataFree.String,
			AutoIncrement:  autoIncrement.String,
			TableCollation: tableCollation.String,
			TableComment:   tableComment.String,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *MySQLHandler) GetMetrics(c *gin.Context) {
	if h.db == nil {
		log.Error().Msg("MySQL connection not available")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "MySQL connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := MetricsResponse{}

	innodbRows, err := h.db.QueryContext(ctx, "SHOW STATUS LIKE 'Innodb%'")
	if err == nil {
		defer innodbRows.Close()
		innodbStats := make(map[string]string)
		for innodbRows.Next() {
			var variable, value string
			if err := innodbRows.Scan(&variable, &value); err == nil {
				innodbStats[variable] = value
			}
		}
		metrics.InnoDB = innodbStats
	}

	// Get global status
	globalRows, err := h.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err == nil {
		defer globalRows.Close()
		globalStats := make(map[string]string)
		for globalRows.Next() {
			var variable, value string
			if err := globalRows.Scan(&variable, &value); err == nil {
				globalStats[variable] = value
			}
		}
		metrics.Global = globalStats
	}

	// Get global variables
	variableRows, err := h.db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err == nil {
		defer variableRows.Close()
		variables := make(map[string]string)
		for variableRows.Next() {
			var variable, value string
			if err := variableRows.Scan(&variable, &value); err == nil {
				variables[variable] = value
			}
		}
		metrics.Variables = variables
	}

	// Get process list (current queries)
	processRows, err := h.db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err == nil {
		defer processRows.Close()
		processList := make([]ProcessInfo, 0)
		for processRows.Next() {
			var id, time int64
			var user, host, command string
			var db, state, info sql.NullString

			err := processRows.Scan(&id, &user, &host, &db, &command, &time, &state, &info)
			if err == nil {
				proc := ProcessInfo{
					ID:      id,
					User:    user,
					Host:    host,
					Command: command,
					Time:    time,
				}
				if db.Valid {
					proc.DB = &db.String
				}
				if state.Valid {
					proc.State = &state.String
				}
				if info.Valid {
					proc.Info = &info.String
				}
				processList = append(processList, proc)
			}
		}
		metrics.ProcessList = processList
	}

	// Get replication status (slave status)
	replicationStatus := h.getReplicationStatus(ctx)
	if replicationStatus != nil {
		metrics.Replication = replicationStatus
	}

	// Extract connection statistics
	connStats := h.extractConnectionStats(metrics.Global, metrics.Variables)
	if connStats != nil {
		metrics.Connections = connStats
	}

	// Get lock information
	lockInfo := h.getLockInfo(ctx, metrics.ProcessList)
	if lockInfo != nil {
		metrics.Locks = lockInfo
	}

	recordMySQLOperation("get_metrics", nil)
	updateMySQLConnectionMetrics(h.db)
	c.JSON(http.StatusOK, metrics)
}

func (h *MySQLHandler) getReplicationStatus(ctx context.Context) *ReplicationStatus {
	// Try to get slave status first
	rows, err := h.db.QueryContext(ctx, "SHOW SLAVE STATUS")
	if err != nil {
		// If slave status fails, try master status
		rows, err = h.db.QueryContext(ctx, "SHOW MASTER STATUS")
		if err != nil {
			return nil
		}
		defer rows.Close()

		// Master status has fewer columns
		if rows.Next() {
			var file string
			var position int64
			var binlogDoDB, binlogIgnoreDB sql.NullString
			var executedGtidSet sql.NullString

			err := rows.Scan(&file, &position, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet)
			if err == nil {
				return &ReplicationStatus{
					MasterLogFile:    &file,
					ReadMasterLogPos: &position,
				}
			}
		}
		return nil
	}
	defer rows.Close()

	if !rows.Next() {
		return nil
	}

	// SHOW SLAVE STATUS returns many columns - use dynamic scanning
	columns, err := rows.Columns()
	if err != nil {
		return nil
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	err = rows.Scan(valuePtrs...)
	if err != nil {
		return nil
	}

	// Map column values to struct
	status := &ReplicationStatus{}
	columnMap := make(map[string]interface{})
	for i, col := range columns {
		if i < len(values) {
			columnMap[col] = values[i]
		}
	}

	// Helper function to convert interface{} to string
	getString := func(key string) *string {
		if val, ok := columnMap[key]; ok && val != nil {
			switch v := val.(type) {
			case string:
				if v != "" {
					return &v
				}
			case []byte:
				if len(v) > 0 {
					s := string(v)
					return &s
				}
			}
		}
		return nil
	}

	// Helper function to convert interface{} to int64
	getInt64 := func(key string) *int64 {
		if val, ok := columnMap[key]; ok && val != nil {
			switch v := val.(type) {
			case int64:
				return &v
			case int32:
				i := int64(v)
				return &i
			case int:
				i := int64(v)
				return &i
			case []byte:
				// Try to parse as int64
				var i int64
				if _, err := fmt.Sscanf(string(v), "%d", &i); err == nil {
					return &i
				}
			}
		}
		return nil
	}

	// Extract key fields
	status.MasterHost = getString("Master_Host")
	status.MasterUser = getString("Master_User")
	if port := getInt64("Master_Port"); port != nil {
		p := int(*port)
		status.MasterPort = &p
	}
	status.SlaveIORunning = getString("Slave_IO_Running")
	status.SlaveSQLRunning = getString("Slave_SQL_Running")
	status.SecondsBehindMaster = getInt64("Seconds_Behind_Master")
	status.MasterLogFile = getString("Master_Log_File")
	status.ReadMasterLogPos = getInt64("Read_Master_Log_Pos")
	status.RelayLogFile = getString("Relay_Log_File")
	status.RelayLogPos = getInt64("Relay_Log_Pos")
	status.LastIOError = getString("Last_IO_Error")
	status.LastSQLError = getString("Last_SQL_Error")

	// Only return if we got some meaningful data
	if status.MasterHost != nil || status.SlaveIORunning != nil {
		return status
	}

	return nil
}

func (h *MySQLHandler) extractConnectionStats(globalStatus, variables map[string]string) *ConnectionStats {
	if globalStatus == nil || variables == nil {
		return nil
	}

	stats := &ConnectionStats{}

	if val, ok := variables["max_connections"]; ok {
		stats.MaxConnections = val
	}
	if val, ok := globalStatus["Max_used_connections"]; ok {
		stats.MaxUsedConnections = val
	}
	if val, ok := globalStatus["Threads_connected"]; ok {
		stats.ThreadsConnected = val
	}
	if val, ok := globalStatus["Threads_running"]; ok {
		stats.ThreadsRunning = val
	}
	if val, ok := globalStatus["Aborted_connects"]; ok {
		stats.AbortedConnects = val
	}
	if val, ok := globalStatus["Connection_errors_max_connections"]; ok {
		stats.ConnectionErrorsMaxConnections = val
	}

	// Only return if we have at least some data
	if stats.MaxConnections != "" || stats.ThreadsConnected != "" {
		return stats
	}

	return nil
}

func (h *MySQLHandler) getLockInfo(ctx context.Context, processList []ProcessInfo) *LockInfo {
	if len(processList) == 0 {
		return nil
	}

	lockInfo := &LockInfo{}

	// Find queries that are waiting (State contains 'Locked' or 'Waiting')
	waitingQueries := make([]ProcessInfo, 0)
	for _, proc := range processList {
		if proc.State != nil {
			state := *proc.State
			if state == "Locked" || state == "Waiting for table metadata lock" ||
				state == "Waiting for lock" || state == "Waiting" {
				waitingQueries = append(waitingQueries, proc)
			}
		}
	}

	if len(waitingQueries) > 0 {
		lockInfo.WaitingQueries = waitingQueries
	}

	// Get InnoDB status (contains lock information)
	var innodbStatus sql.NullString
	err := h.db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS").Scan(&innodbStatus)
	if err == nil && innodbStatus.Valid {
		status := innodbStatus.String
		lockInfo.InnoDBStatus = &status

		// Extract deadlock information if present
		if contains(status, "DEADLOCK") || contains(status, "deadlock") {
			lockInfo.Deadlocks = &status
		}
	}

	// Only return if we have meaningful data
	if len(lockInfo.WaitingQueries) > 0 || lockInfo.InnoDBStatus != nil {
		return lockInfo
	}

	return nil
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
