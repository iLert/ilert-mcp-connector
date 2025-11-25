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
