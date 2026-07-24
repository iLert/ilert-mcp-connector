package main

// Kafka response types

type TopicInfo struct {
	Name       string  `json:"name"`
	Partitions int     `json:"partitions"`
	Error      *string `json:"error,omitempty"`
}

type TopicsResponse struct {
	Topics []TopicInfo `json:"topics"`
}

type PartitionInfo struct {
	ID       int   `json:"id"`
	Leader   int   `json:"leader"`
	Replicas []int `json:"replicas"`
	ISR      []int `json:"isr"`
}

type TopicDetailResponse struct {
	Name       string          `json:"name"`
	Partitions []PartitionInfo `json:"partitions"`
	Error      *string         `json:"error,omitempty"`
}

type ConsumerGroupInfo struct {
	GroupID string `json:"groupId"`
}

type ConsumersResponse struct {
	Consumers []ConsumerGroupInfo `json:"consumers"`
}

type OwnedPartitionInfo struct {
	Topic      string `json:"topic"`
	Partitions []int  `json:"partitions"`
}

type MemberMetadataInfo struct {
	Version         int                  `json:"version"`
	Topics          []string             `json:"topics"`
	UserData        []byte               `json:"userData,omitempty"`
	OwnedPartitions []OwnedPartitionInfo `json:"ownedPartitions,omitempty"`
}

type MemberInfo struct {
	MemberID       string              `json:"memberId"`
	ClientID       string              `json:"clientId"`
	ClientHost     string              `json:"clientHost"`
	MemberMetadata *MemberMetadataInfo `json:"memberMetadata,omitempty"`
}

type ConsumerGroupDetailResponse struct {
	GroupID string       `json:"groupId"`
	Members []MemberInfo `json:"members"`
	Error   *string      `json:"error,omitempty"`
}

type ConsumerLagInfo struct {
	Topic         string `json:"topic"`
	Partition     int    `json:"partition"`
	CurrentOffset int64  `json:"currentOffset"`
	HighWaterMark int64  `json:"highWaterMark"`
	Lag           int64  `json:"lag"`
}

type ConsumerLagResponse struct {
	GroupID string            `json:"groupId"`
	Lag     []ConsumerLagInfo `json:"lag"`
}

// MySQL response types

type DatabasesResponse struct {
	Databases []string `json:"databases"`
}

type TablesResponse struct {
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
}

type ColumnInfo struct {
	Field   string  `json:"field"`
	Type    string  `json:"type"`
	Null    string  `json:"null"`
	Key     string  `json:"key"`
	Default *string `json:"default,omitempty"`
	Extra   *string `json:"extra,omitempty"`
}

type TableStatus struct {
	Engine         string `json:"engine"`
	RowFormat      string `json:"rowFormat"`
	TableRows      string `json:"tableRows"`
	AvgRowLength   string `json:"avgRowLength"`
	DataLength     string `json:"dataLength"`
	IndexLength    string `json:"indexLength"`
	DataFree       string `json:"dataFree"`
	AutoIncrement  string `json:"autoIncrement"`
	TableCollation string `json:"tableCollation"`
	TableComment   string `json:"tableComment"`
}

type TableDetailResponse struct {
	Database string       `json:"database"`
	Table    string       `json:"table"`
	Columns  []ColumnInfo `json:"columns"`
	Status   *TableStatus `json:"status,omitempty"`
}

type ProcessInfo struct {
	ID      int64   `json:"id"`
	User    string  `json:"user"`
	Host    string  `json:"host"`
	DB      *string `json:"db,omitempty"`
	Command string  `json:"command"`
	Time    int64   `json:"time"`
	State   *string `json:"state,omitempty"`
	Info    *string `json:"info,omitempty"`
}

type ReplicationStatus struct {
	MasterHost          *string `json:"masterHost,omitempty"`
	MasterUser          *string `json:"masterUser,omitempty"`
	MasterPort          *int    `json:"masterPort,omitempty"`
	SlaveIORunning      *string `json:"slaveIORunning,omitempty"`
	SlaveSQLRunning     *string `json:"slaveSQLRunning,omitempty"`
	SecondsBehindMaster *int64  `json:"secondsBehindMaster,omitempty"`
	MasterLogFile       *string `json:"masterLogFile,omitempty"`
	ReadMasterLogPos    *int64  `json:"readMasterLogPos,omitempty"`
	RelayLogFile        *string `json:"relayLogFile,omitempty"`
	RelayLogPos         *int64  `json:"relayLogPos,omitempty"`
	LastIOError         *string `json:"lastIOError,omitempty"`
	LastSQLError        *string `json:"lastSQLError,omitempty"`
}

type ConnectionStats struct {
	MaxConnections                 string `json:"maxConnections"`
	MaxUsedConnections             string `json:"maxUsedConnections"`
	ThreadsConnected               string `json:"threadsConnected"`
	ThreadsRunning                 string `json:"threadsRunning"`
	AbortedConnects                string `json:"abortedConnects"`
	ConnectionErrorsMaxConnections string `json:"connectionErrorsMaxConnections"`
}

type LockInfo struct {
	WaitingQueries []ProcessInfo `json:"waitingQueries,omitempty"`
	InnoDBStatus   *string       `json:"innodbStatus,omitempty"`
	Deadlocks      *string       `json:"deadlocks,omitempty"`
}

type MetricsResponse struct {
	InnoDB      map[string]string  `json:"innodb,omitempty"`
	Global      map[string]string  `json:"global,omitempty"`
	Variables   map[string]string  `json:"variables,omitempty"`
	ProcessList []ProcessInfo      `json:"processList,omitempty"`
	Replication *ReplicationStatus `json:"replication,omitempty"`
	Connections *ConnectionStats   `json:"connections,omitempty"`
	Locks       *LockInfo          `json:"locks,omitempty"`
}

// PostgreSQL response types

type PostgresColumnInfo struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Nullable   string  `json:"nullable"`
	Default    *string `json:"default,omitempty"`
	MaxLength  *int64  `json:"maxLength,omitempty"`
	Position   int64   `json:"position"`
	IsIdentity string  `json:"isIdentity"`
}

type PostgresTableStats struct {
	LiveTuples       int64   `json:"liveTuples"`
	DeadTuples       int64   `json:"deadTuples"`
	SeqScans         int64   `json:"seqScans"`
	IndexScans       int64   `json:"indexScans"`
	LastVacuum       *string `json:"lastVacuum,omitempty"`
	LastAutovacuum   *string `json:"lastAutovacuum,omitempty"`
	LastAnalyze      *string `json:"lastAnalyze,omitempty"`
	LastAutoanalyze  *string `json:"lastAutoanalyze,omitempty"`
	TotalSizeBytes   int64   `json:"totalSizeBytes"`
	TableSizeBytes   int64   `json:"tableSizeBytes"`
	IndexesSizeBytes int64   `json:"indexesSizeBytes"`
	EstimatedRows    float64 `json:"estimatedRows"`
}

type PostgresTableDetailResponse struct {
	Database string               `json:"database"`
	Schema   string               `json:"schema"`
	Table    string               `json:"table"`
	Columns  []PostgresColumnInfo `json:"columns"`
	Stats    *PostgresTableStats  `json:"stats,omitempty"`
}

type ApplicationConnections struct {
	ApplicationName string           `json:"applicationName"`
	Count           int64            `json:"count"`
	States          map[string]int64 `json:"states"`
}

type UserConnections struct {
	User  string `json:"user"`
	Count int64  `json:"count"`
}

type DatabaseConnections struct {
	Database string `json:"database"`
	Count    int64  `json:"count"`
}

type PostgresConnectionStats struct {
	Total                int64                    `json:"total"`
	MaxConnections       int64                    `json:"maxConnections"`
	Available            int64                    `json:"available"`
	ReservedForSuperuser int64                    `json:"reservedForSuperuser"`
	ByApplication        []ApplicationConnections `json:"byApplication"`
	ByUser               []UserConnections        `json:"byUser"`
	ByDatabase           []DatabaseConnections    `json:"byDatabase"`
	ByState              map[string]int64         `json:"byState"`
}

type PgProcessInfo struct {
	PID             int64   `json:"pid"`
	User            *string `json:"user,omitempty"`
	Database        *string `json:"database,omitempty"`
	ApplicationName *string `json:"applicationName,omitempty"`
	ClientAddr      *string `json:"clientAddr,omitempty"`
	State           *string `json:"state,omitempty"`
	BackendStart    *string `json:"backendStart,omitempty"`
	XactStart       *string `json:"xactStart,omitempty"`
	QueryStart      *string `json:"queryStart,omitempty"`
	WaitEventType   *string `json:"waitEventType,omitempty"`
	WaitEvent       *string `json:"waitEvent,omitempty"`
	Query           *string `json:"query,omitempty"`
}

type PgDatabaseStats struct {
	Database      string  `json:"database"`
	XactCommit    int64   `json:"xactCommit"`
	XactRollback  int64   `json:"xactRollback"`
	BlksRead      int64   `json:"blksRead"`
	BlksHit       int64   `json:"blksHit"`
	CacheHitRatio float64 `json:"cacheHitRatio"`
	TupReturned   int64   `json:"tupReturned"`
	TupFetched    int64   `json:"tupFetched"`
	TupInserted   int64   `json:"tupInserted"`
	TupUpdated    int64   `json:"tupUpdated"`
	TupDeleted    int64   `json:"tupDeleted"`
	Conflicts     int64   `json:"conflicts"`
	Deadlocks     int64   `json:"deadlocks"`
	TempFiles     int64   `json:"tempFiles"`
	TempBytes     int64   `json:"tempBytes"`
}

type PgReplicaInfo struct {
	ClientAddr      *string `json:"clientAddr,omitempty"`
	ApplicationName string  `json:"applicationName"`
	State           string  `json:"state"`
	SentLsn         *string `json:"sentLsn,omitempty"`
	ReplayLsn       *string `json:"replayLsn,omitempty"`
	WriteLag        *string `json:"writeLag,omitempty"`
	FlushLag        *string `json:"flushLag,omitempty"`
	ReplayLag       *string `json:"replayLag,omitempty"`
}

type PgReplicationInfo struct {
	IsInRecovery  bool            `json:"isInRecovery"`
	Replicas      []PgReplicaInfo `json:"replicas,omitempty"`
	LastReplayLsn *string         `json:"lastReplayLsn,omitempty"`
	ReplayLagSecs *float64        `json:"replayLagSecs,omitempty"`
}

type PgBlockedQuery struct {
	BlockedPID   int64   `json:"blockedPid"`
	BlockedQuery *string `json:"blockedQuery,omitempty"`
	BlockedUser  *string `json:"blockedUser,omitempty"`
	BlockingPIDs []int64 `json:"blockingPids"`
}

type PgLockInfo struct {
	BlockedQueries []PgBlockedQuery `json:"blockedQueries,omitempty"`
	TotalLocks     int64            `json:"totalLocks"`
	UngrantedLocks int64            `json:"ungrantedLocks"`
}

type PgBgWriterStats struct {
	CheckpointsTimed     *int64 `json:"checkpointsTimed,omitempty"`
	CheckpointsRequested *int64 `json:"checkpointsRequested,omitempty"`
	BuffersCheckpoint    *int64 `json:"buffersCheckpoint,omitempty"`
	BuffersClean         *int64 `json:"buffersClean,omitempty"`
	BuffersBackend       *int64 `json:"buffersBackend,omitempty"`
	BuffersAlloc         *int64 `json:"buffersAlloc,omitempty"`
}

type PgVacuumTableInfo struct {
	Schema         string  `json:"schema"`
	Table          string  `json:"table"`
	DeadTuples     int64   `json:"deadTuples"`
	LiveTuples     int64   `json:"liveTuples"`
	LastVacuum     *string `json:"lastVacuum,omitempty"`
	LastAutovacuum *string `json:"lastAutovacuum,omitempty"`
}

type PgVacuumInfo struct {
	TablesByDeadTuples []PgVacuumTableInfo `json:"tablesByDeadTuples,omitempty"`
	OldestXidAge       *int64              `json:"oldestXidAge,omitempty"`
}

type PgSessionIssue struct {
	PID             int64   `json:"pid"`
	User            *string `json:"user,omitempty"`
	Database        *string `json:"database,omitempty"`
	ApplicationName *string `json:"applicationName,omitempty"`
	ClientAddr      *string `json:"clientAddr,omitempty"`
	State           *string `json:"state,omitempty"`
	Query           *string `json:"query,omitempty"`
	DurationSecs    float64 `json:"durationSecs"`
}

type PgSessionIssues struct {
	ThresholdSecs      int64            `json:"thresholdSecs"`
	LongRunningQueries []PgSessionIssue `json:"longRunningQueries"`
	IdleInTransaction  []PgSessionIssue `json:"idleInTransaction"`
}

type PostgresMetricsResponse struct {
	Activity      []PgProcessInfo          `json:"activity,omitempty"`
	DatabaseStats []PgDatabaseStats        `json:"databaseStats,omitempty"`
	Settings      map[string]string        `json:"settings,omitempty"`
	Replication   *PgReplicationInfo       `json:"replication,omitempty"`
	Connections   *PostgresConnectionStats `json:"connections,omitempty"`
	Locks         *PgLockInfo              `json:"locks,omitempty"`
	BgWriter      *PgBgWriterStats         `json:"bgwriter,omitempty"`
	Vacuum        *PgVacuumInfo            `json:"vacuum,omitempty"`
	Sessions      *PgSessionIssues         `json:"sessions,omitempty"`
}

// Error response type

type ErrorResponse struct {
	Error string `json:"error"`
}

// Version response type

type VersionResponse struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// ClickHouse response types

type ClickHouseColumnInfo struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	DefaultKind *string `json:"defaultKind,omitempty"`
	DefaultExpr *string `json:"defaultExpr,omitempty"`
	Comment     *string `json:"comment,omitempty"`
	CodecExpr   *string `json:"codecExpr,omitempty"`
	TTLExpr     *string `json:"ttlExpr,omitempty"`
}

type ClickHouseTableInfo struct {
	Database     string `json:"database"`
	Table        string `json:"table"`
	Engine       string `json:"engine"`
	TotalRows    uint64 `json:"totalRows"`
	TotalBytes   uint64 `json:"totalBytes"`
	PrimaryKey   string `json:"primaryKey"`
	SortingKey   string `json:"sortingKey"`
	PartitionKey string `json:"partitionKey"`
}

type ClickHouseMetrics struct {
	Metrics             map[string]float64   `json:"metrics,omitempty"`
	Events              map[string]uint64    `json:"events,omitempty"`
	AsynchronousMetrics map[string]float64   `json:"asynchronousMetrics,omitempty"`
	Replicas            []ClickHouseReplica  `json:"replicas,omitempty"`
	Processes           []ClickHouseProcess  `json:"processes,omitempty"`
	Merges              []ClickHouseMerge    `json:"merges,omitempty"`
	Mutations           []ClickHouseMutation `json:"mutations,omitempty"`
}

type ClickHouseReplica struct {
	Database         string `json:"database"`
	Table            string `json:"table"`
	IsLeader         uint8  `json:"isLeader"`
	IsReadonly       uint8  `json:"isReadonly"`
	IsSessionExpired uint8  `json:"isSessionExpired"`
	FutureParts      uint64 `json:"futureParts"`
	PartsToCheck     uint64 `json:"partsToCheck"`
	ZookeeperPath    string `json:"zookeeperPath"`
	ReplicaPath      string `json:"replicaPath"`
	ColumnsVersion   int64  `json:"columnsVersion"`
	QueueSize        uint64 `json:"queueSize"`
	InsertsInQueue   uint64 `json:"insertsInQueue"`
	MergesInQueue    uint64 `json:"mergesInQueue"`
	LogMaxIndex      uint64 `json:"logMaxIndex"`
	LogPointer       uint64 `json:"logPointer"`
	TotalReplicas    uint8  `json:"totalReplicas"`
	ActiveReplicas   uint8  `json:"activeReplicas"`
	LostPartCount    uint64 `json:"lostPartCount"`
}

type ClickHouseProcess struct {
	QueryID      string  `json:"queryId"`
	User         string  `json:"user"`
	Address      string  `json:"address"`
	Elapsed      float64 `json:"elapsed"`
	ReadRows     uint64  `json:"readRows"`
	ReadBytes    uint64  `json:"readBytes"`
	TotalRows    uint64  `json:"totalRows"`
	WrittenRows  uint64  `json:"writtenRows"`
	WrittenBytes uint64  `json:"writtenBytes"`
	MemoryUsage  uint64  `json:"memoryUsage"`
	Query        string  `json:"query"`
}

type ClickHouseMerge struct {
	Database        string  `json:"database"`
	Table           string  `json:"table"`
	Elapsed         float64 `json:"elapsed"`
	Progress        uint64  `json:"progress"`
	NumPartsToMerge uint64  `json:"numPartsToMerge"`
	RowsRead        uint64  `json:"rowsRead"`
	BytesRead       uint64  `json:"bytesRead"`
	RowsWritten     uint64  `json:"rowsWritten"`
	BytesWritten    uint64  `json:"bytesWritten"`
	MemoryUsage     uint64  `json:"memoryUsage"`
}

type ClickHouseMutation struct {
	Database         string  `json:"database"`
	Table            string  `json:"table"`
	MutationID       string  `json:"mutationId"`
	Command          string  `json:"command"`
	CreateTime       string  `json:"createTime"`
	BlockNumbers     string  `json:"blockNumbers"`
	PartsToDo        uint64  `json:"partsToDo"`
	IsDone           uint8   `json:"isDone"`
	LatestFailedPart *string `json:"latestFailedPart,omitempty"`
	LatestFailTime   *string `json:"latestFailTime,omitempty"`
	LatestFailReason *string `json:"latestFailReason,omitempty"`
}

// Redis response types

type KeysResponse struct {
	Keys    []string `json:"keys"`
	Count   int      `json:"count"`
	Pattern string   `json:"pattern,omitempty"`
	Cursor  uint64   `json:"cursor"`
	HasMore bool     `json:"hasMore"`
}

type KeyInfo struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	TTL      int64   `json:"ttl"`
	Size     int64   `json:"size"`
	Encoding string  `json:"encoding,omitempty"`
	Error    *string `json:"error,omitempty"`
}

type KeyInfoResponse struct {
	Info KeyInfo `json:"info"`
}

type RedisInfoResponse struct {
	Info  map[string]string `json:"info"`
	Error *string           `json:"error,omitempty"`
}

type RedisDatabasesResponse struct {
	Databases []int `json:"databases"`
}

type RedisMetrics struct {
	Server      map[string]string `json:"server,omitempty"`
	Memory      map[string]string `json:"memory,omitempty"`
	Stats       map[string]string `json:"stats,omitempty"`
	Clients     map[string]string `json:"clients,omitempty"`
	Persistence map[string]string `json:"persistence,omitempty"`
	Replication map[string]string `json:"replication,omitempty"`
	CPU         map[string]string `json:"cpu,omitempty"`
	Keyspace    map[string]string `json:"keyspace,omitempty"`
}
