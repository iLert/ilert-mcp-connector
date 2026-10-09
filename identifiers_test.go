package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
)

func TestValidIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "users", true},
		{"underscore and digits", "app_db_2", true},
		{"hyphen", "my-app", true},
		{"dollar", "tbl$1", true},
		{"clickhouse inner table", ".inner.mv_events", true},
		{"max length", strings.Repeat("a", 128), true},
		{"empty", "", false},
		{"too long", strings.Repeat("a", 129), false},
		{"single quote", "db'name", false},
		{"double quote", `db"name`, false},
		{"backtick", "db`name", false},
		{"space", "db name", false},
		{"semicolon", "db;name", false},
		{"backslash", `db\name`, false},
		{"slash", "db/name", false},
		{"parenthesis", "db(name)", false},
		{"newline", "db\nname", false},
		{"non-ascii", "datenbänk", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validIdentifier(tt.input); got != tt.want {
				t.Errorf("validIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func newIdentifierTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Neither driver connects on open, so no server is needed as long as
	// requests are rejected before any query is issued.
	mysqlDB, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:1)/db")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() {
		mysqlDB.Close()
	})

	chConn, err := clickhouse.Open(&clickhouse.Options{Addr: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("clickhouse.Open: %v", err)
	}
	t.Cleanup(func() {
		chConn.Close()
	})

	mysqlHandler := &MySQLHandler{db: mysqlDB}
	chHandler := &ClickHouseHandler{conn: chConn}

	router := gin.New()
	router.GET("/mysql/databases/:database/tables", mysqlHandler.GetTables)
	router.GET("/mysql/databases/:database/tables/:table", mysqlHandler.DescribeTable)
	router.GET("/clickhouse/databases/:database/tables", chHandler.GetTables)
	router.GET("/clickhouse/databases/:database/tables/:table", chHandler.DescribeTable)
	return router
}

func TestHandlersRejectInvalidIdentifiers(t *testing.T) {
	router := newIdentifierTestRouter(t)

	tests := []struct {
		name      string
		path      string
		wantError string
	}{
		{"mysql tables bad database", "/mysql/databases/db%27x/tables", "invalid database name"},
		{"mysql describe bad database", "/mysql/databases/db%60x/tables/users", "invalid database name"},
		{"mysql describe bad table", "/mysql/databases/app/tables/users%27x", "invalid table name"},
		{"clickhouse tables bad database", "/clickhouse/databases/db%27x/tables", "invalid database name"},
		{"clickhouse describe bad database", "/clickhouse/databases/db%20x/tables/events", "invalid database name"},
		{"clickhouse describe bad table", "/clickhouse/databases/default/tables/events%3Bx", "invalid table name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("GET %s = %d, want %d", tt.path, w.Code, http.StatusBadRequest)
			}
			var body ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error != tt.wantError {
				t.Errorf("GET %s error = %q, want %q", tt.path, body.Error, tt.wantError)
			}
		})
	}
}
