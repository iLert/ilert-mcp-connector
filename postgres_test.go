package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSplitSchemaTable(t *testing.T) {
	tests := []struct {
		name       string
		param      string
		wantSchema string
		wantTable  string
	}{
		{"bare table defaults to public", "users", "public", "users"},
		{"schema qualified", "audit.events", "audit", "events"},
		{"leading dot keeps public", ".users", "public", ".users"},
		{"nested dots split on first", "a.b.c", "a", "b.c"},
		{"empty string", "", "public", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table := splitSchemaTable(tt.param)
			if schema != tt.wantSchema || table != tt.wantTable {
				t.Errorf("splitSchemaTable(%q) = (%q, %q), want (%q, %q)", tt.param, schema, table, tt.wantSchema, tt.wantTable)
			}
		})
	}
}

func TestParsePgIntArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int64
	}{
		{"empty array", "{}", []int64{}},
		{"single element", "{42}", []int64{42}},
		{"multiple elements", "{1,2,3}", []int64{1, 2, 3}},
		{"with spaces", "{1, 2, 3}", []int64{1, 2, 3}},
		{"skips invalid", "{1,abc,3}", []int64{1, 3}},
		{"empty string", "", []int64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePgIntArray(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePgIntArray(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPostgresDatabaseNamePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple name", "mydb", true},
		{"with underscore and dash", "my_db-1", true},
		{"uppercase", "MyDB", true},
		{"empty", "", false},
		{"sslmode injection", "db?sslmode=require", false},
		{"path traversal", "../etc", false},
		{"space", "my db", false},
		{"quote", "db'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgresDatabaseNamePattern.MatchString(tt.input); got != tt.want {
				t.Errorf("postgresDatabaseNamePattern.MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPostgresDSN(t *testing.T) {
	config := PostgresConfig{
		Host:     "db.example.com",
		Port:     "5432",
		User:     "app",
		Password: "p@ss:word/1",
		SSLMode:  "require",
	}

	dsn := buildPostgresDSN(config, "orders")
	want := "postgres://app:p%40ss%3Aword%2F1@db.example.com:5432/orders?sslmode=require"
	if dsn != want {
		t.Errorf("buildPostgresDSN() = %q, want %q", dsn, want)
	}
}

func TestParseSessionThreshold(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{"empty uses default", "", 60},
		{"valid value", "300", 300},
		{"one second", "1", 1},
		{"zero uses default", "0", 60},
		{"negative uses default", "-5", 60},
		{"garbage uses default", "abc", 60},
		{"float uses default", "1.5", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSessionThreshold(tt.input); got != tt.want {
				t.Errorf("parseSessionThreshold(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeCacheHitRatio(t *testing.T) {
	tests := []struct {
		name string
		hit  int64
		read int64
		want float64
	}{
		{"zero total", 0, 0, 0},
		{"all hits", 100, 0, 1},
		{"all reads", 0, 100, 0},
		{"half", 50, 50, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeCacheHitRatio(tt.hit, tt.read); got != tt.want {
				t.Errorf("computeCacheHitRatio(%d, %d) = %v, want %v", tt.hit, tt.read, got, tt.want)
			}
		})
	}
}

func TestAggregatePostgresConnections(t *testing.T) {
	groups := []pgConnGroup{
		{App: "workforce", User: "app", Database: "orders", State: "active", Count: 3},
		{App: "workforce", User: "app", Database: "orders", State: "idle", Count: 5},
		{App: "hive", User: "app", Database: "orders", State: "active", Count: 2},
		{App: "", User: "postgres", Database: "postgres", State: "idle", Count: 1},
	}

	stats := aggregatePostgresConnections(groups, 100, 3)

	if stats.Total != 11 {
		t.Errorf("Total = %d, want 11", stats.Total)
	}
	if stats.MaxConnections != 100 {
		t.Errorf("MaxConnections = %d, want 100", stats.MaxConnections)
	}
	if stats.ReservedForSuperuser != 3 {
		t.Errorf("ReservedForSuperuser = %d, want 3", stats.ReservedForSuperuser)
	}
	if stats.Available != 86 {
		t.Errorf("Available = %d, want 86", stats.Available)
	}

	if len(stats.ByApplication) != 3 {
		t.Fatalf("len(ByApplication) = %d, want 3", len(stats.ByApplication))
	}
	if stats.ByApplication[0].ApplicationName != "workforce" || stats.ByApplication[0].Count != 8 {
		t.Errorf("ByApplication[0] = %+v, want workforce with count 8", stats.ByApplication[0])
	}
	if stats.ByApplication[0].States["active"] != 3 || stats.ByApplication[0].States["idle"] != 5 {
		t.Errorf("ByApplication[0].States = %v, want active:3 idle:5", stats.ByApplication[0].States)
	}
	if stats.ByApplication[1].ApplicationName != "hive" {
		t.Errorf("ByApplication[1] = %+v, want hive", stats.ByApplication[1])
	}
	if stats.ByApplication[2].ApplicationName != "" || stats.ByApplication[2].Count != 1 {
		t.Errorf("ByApplication[2] = %+v, want empty app name with count 1", stats.ByApplication[2])
	}

	if len(stats.ByUser) != 2 || stats.ByUser[0].User != "app" || stats.ByUser[0].Count != 10 {
		t.Errorf("ByUser = %+v, want app:10 first", stats.ByUser)
	}

	if len(stats.ByDatabase) != 2 || stats.ByDatabase[0].Database != "orders" || stats.ByDatabase[0].Count != 10 {
		t.Errorf("ByDatabase = %+v, want orders:10 first", stats.ByDatabase)
	}

	if stats.ByState["active"] != 5 || stats.ByState["idle"] != 6 {
		t.Errorf("ByState = %v, want active:5 idle:6", stats.ByState)
	}
}

func TestAggregatePostgresConnectionsEmpty(t *testing.T) {
	stats := aggregatePostgresConnections([]pgConnGroup{}, 0, 0)

	if stats.Total != 0 {
		t.Errorf("Total = %d, want 0", stats.Total)
	}
	if stats.Available != 0 {
		t.Errorf("Available = %d, want 0", stats.Available)
	}
	if len(stats.ByApplication) != 0 || len(stats.ByUser) != 0 || len(stats.ByDatabase) != 0 {
		t.Errorf("expected empty breakdowns, got %+v", stats)
	}
}

func TestAggregatePostgresConnectionsAvailableFloor(t *testing.T) {
	groups := []pgConnGroup{
		{App: "a", User: "u", Database: "d", State: "active", Count: 10},
	}

	stats := aggregatePostgresConnections(groups, 5, 3)

	if stats.Available != 0 {
		t.Errorf("Available = %d, want 0 (floored)", stats.Available)
	}
}

func TestPostgresHandlersWithoutConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &PostgresHandler{config: PostgresConfig{}, db: nil}

	router := gin.New()
	router.GET("/postgres/databases", handler.GetDatabases)
	router.GET("/postgres/databases/:database/tables", handler.GetTables)
	router.GET("/postgres/databases/:database/tables/:table", handler.DescribeTable)
	router.GET("/postgres/metrics", handler.GetMetrics)

	paths := []string{
		"/postgres/databases",
		"/postgres/databases/mydb/tables",
		"/postgres/databases/mydb/tables/users",
		"/postgres/metrics",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("GET %s = %d, want %d", path, w.Code, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestGetTablesInvalidDatabaseName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stubConfig := PostgresConfig{
		Host:     "127.0.0.1",
		Port:     "1",
		User:     "stub",
		Password: "stub",
		Database: "main",
		SSLMode:  "disable",
	}
	db, err := openPostgresDB(stubConfig, stubConfig.Database)
	if err != nil {
		t.Fatalf("failed to open stub db: %v", err)
	}
	defer db.Close()

	handler := &PostgresHandler{
		config: stubConfig,
		db:     db,
	}

	router := gin.New()
	router.GET("/postgres/databases/:database/tables", handler.GetTables)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/postgres/databases/bad%20name/tables", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET with invalid database name = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
