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
	Key        string `json:"key"`
	Type       string `json:"type"`
	TTL        int64  `json:"ttl"`
	Size       int64  `json:"size"`
	Encoding   string `json:"encoding,omitempty"`
	Error      *string `json:"error,omitempty"`
}

type KeyInfoResponse struct {
	Info KeyInfo `json:"info"`
}

type RedisInfoResponse struct {
	Info map[string]string `json:"info"`
	Error *string          `json:"error,omitempty"`
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
