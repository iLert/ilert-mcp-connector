package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidMySQLIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "users", true},
		{"underscore and digits", "app_db_2", true},
		{"hyphen", "my-app", true},
		{"dollar", "tbl$1", true},
		{"dot", "a.b", true},
		{"non-ascii", "datenbänk", true},
		{"cjk", "数据库", true},
		{"inner space", "db name", true},
		{"single quote", "db'name", true},
		{"backtick", "db`name", true},
		{"semicolon", "db;name", true},
		{"leading space", " db", true},
		{"max length", strings.Repeat("a", 64), true},
		{"max length multibyte", strings.Repeat("ä", 64), true},
		{"empty", "", false},
		{"too long", strings.Repeat("a", 65), false},
		{"too long multibyte", strings.Repeat("ä", 65), false},
		{"trailing space", "db ", false},
		{"only space", " ", false},
		{"nul", "db\x00name", false},
		{"supplementary character", "db😀", false},
		{"invalid utf-8", "db\xffname", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validMySQLIdentifier(tt.input); got != tt.want {
				t.Errorf("validMySQLIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuoteMySQLIdentifier(t *testing.T) {
	tests := map[string]string{
		"users":          "`users`",
		"db`name":        "`db``name`",
		"x`; DROP x; --": "`x``; DROP x; --`",
		"datenbänk":      "`datenbänk`",
	}

	for input, want := range tests {
		if got := quoteMySQLIdentifier(input); got != want {
			t.Errorf("quoteMySQLIdentifier(%q) = %q, want %q", input, got, want)
		}
	}
}

func newIdentifierTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// The driver does not connect on open, so no server is needed as long as
	// requests are rejected before any query is issued.
	mysqlDB, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:1)/db")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() {
		mysqlDB.Close()
	})

	mysqlHandler := &MySQLHandler{db: mysqlDB}

	router := gin.New()
	router.GET("/mysql/databases/:database/tables", mysqlHandler.GetTables)
	router.GET("/mysql/databases/:database/tables/:table", mysqlHandler.DescribeTable)
	return router
}

func TestHandlersRejectInvalidIdentifiers(t *testing.T) {
	router := newIdentifierTestRouter(t)

	tests := []struct {
		name      string
		path      string
		wantError string
	}{
		{"mysql tables trailing space", "/mysql/databases/db%20/tables", "invalid database name"},
		{"mysql describe nul database", "/mysql/databases/db%00x/tables/users", "invalid database name"},
		{"mysql describe supplementary table", "/mysql/databases/app/tables/users%F0%9F%98%80", "invalid table name"},
		{"mysql describe too long table", "/mysql/databases/app/tables/" + strings.Repeat("a", 65), "invalid table name"},
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
