package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
)

type recordedQuery struct {
	query string
	args  []any
}

// fakeClickHouseConn records every query and answers with empty results. Methods the handlers
// under test do not call are left to the embedded nil interface and would panic if used.
type fakeClickHouseConn struct {
	driver.Conn
	queries []recordedQuery
}

func (f *fakeClickHouseConn) Query(_ context.Context, query string, args ...any) (driver.Rows, error) {
	f.queries = append(f.queries, recordedQuery{query: query, args: args})
	return emptyRows{}, nil
}

func (f *fakeClickHouseConn) QueryRow(_ context.Context, query string, args ...any) driver.Row {
	f.queries = append(f.queries, recordedQuery{query: query, args: args})
	return emptyRow{}
}

type emptyRows struct {
	driver.Rows
}

func (emptyRows) Next() bool {
	return false
}

func (emptyRows) Err() error {
	return nil
}

func (emptyRows) Close() error {
	return nil
}

type emptyRow struct {
	driver.Row
}

func (emptyRow) Scan(...any) error {
	return nil
}

func (emptyRow) Err() error {
	return nil
}

// Names ClickHouse accepts as quoted identifiers, plus SQL-looking ones that must stay plain values.
var clickHouseTestNames = []string{
	"events",
	"数据库",
	"订单 表",
	"pay$v2",
	"$$x$$",
	"$1",
	"{p:String}",
	"x?y",
	"shop😀",
	"x' OR '1'='1",
	"x`; DROP DATABASE canary; --",
	`x\' OR 1=1 --`,
}

func newClickHouseTestRouter(conn *fakeClickHouseConn) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := &ClickHouseHandler{conn: conn}
	router := gin.New()
	router.GET("/clickhouse/databases/:database/tables", handler.GetTables)
	router.GET("/clickhouse/databases/:database/tables/:table", handler.DescribeTable)
	return router
}

func serveClickHouse(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want %d: %s", path, w.Code, http.StatusOK, w.Body.String())
	}
	return w
}

// assertBoundOnly checks that the query was issued with exactly the given arguments and that none of
// them was formatted into the SQL text.
func assertBoundOnly(t *testing.T, got recordedQuery, wantArgs ...any) {
	t.Helper()

	if !reflect.DeepEqual(got.args, wantArgs) {
		t.Errorf("args = %#v, want %#v", got.args, wantArgs)
	}
	if strings.Count(got.query, "?") != len(wantArgs) {
		t.Errorf("query has %d placeholders, want %d: %s", strings.Count(got.query, "?"), len(wantArgs), got.query)
	}
	for _, arg := range wantArgs {
		if strings.Contains(got.query, arg.(string)) {
			t.Errorf("query contains the value %q: %s", arg, got.query)
		}
	}
}

func TestClickHouseGetTablesBindsDatabaseName(t *testing.T) {
	for _, database := range clickHouseTestNames {
		t.Run(database, func(t *testing.T) {
			conn := &fakeClickHouseConn{}
			router := newClickHouseTestRouter(conn)

			w := serveClickHouse(t, router, "/clickhouse/databases/"+url.PathEscape(database)+"/tables")

			if len(conn.queries) != 1 {
				t.Fatalf("got %d queries, want 1", len(conn.queries))
			}
			assertBoundOnly(t, conn.queries[0], database)

			var body TablesResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Database != database {
				t.Errorf("response database = %q, want %q", body.Database, database)
			}
			if body.Tables == nil || len(body.Tables) != 0 {
				t.Errorf("response tables = %#v, want empty list", body.Tables)
			}
		})
	}
}

func TestClickHouseDescribeTableBindsNames(t *testing.T) {
	for _, name := range clickHouseTestNames {
		t.Run(name, func(t *testing.T) {
			conn := &fakeClickHouseConn{}
			router := newClickHouseTestRouter(conn)
			database := "db " + name
			table := name

			w := serveClickHouse(t, router, "/clickhouse/databases/"+url.PathEscape(database)+"/tables/"+url.PathEscape(table))

			if len(conn.queries) != 2 {
				t.Fatalf("got %d queries, want 2 (columns, table info)", len(conn.queries))
			}
			for _, q := range conn.queries {
				assertBoundOnly(t, q, database, table)
			}

			var body struct {
				Database string `json:"database"`
				Table    string `json:"table"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Database != database || body.Table != table {
				t.Errorf("response = %q.%q, want %q.%q", body.Database, body.Table, database, table)
			}
		})
	}
}

func TestValidClickHouseName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "events", true},
		{"255 bytes ascii", strings.Repeat("a", 255), true},
		{"255 bytes multibyte", strings.Repeat("ä", 127) + "a", true},
		{"empty", "", false},
		{"256 bytes ascii", strings.Repeat("a", 256), false},
		{"256 bytes multibyte", strings.Repeat("ä", 128), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validClickHouseName(tt.input); got != tt.want {
				t.Errorf("validClickHouseName(%d bytes) = %v, want %v", len(tt.input), got, tt.want)
			}
		})
	}
}

func TestClickHouseHandlersRejectTooLongNames(t *testing.T) {
	tooLong := url.PathEscape(strings.Repeat("ä", 128))

	tests := []struct {
		name      string
		path      string
		wantError string
	}{
		{"tables long database", "/clickhouse/databases/" + tooLong + "/tables", "invalid database name"},
		{"describe long database", "/clickhouse/databases/" + tooLong + "/tables/events", "invalid database name"},
		{"describe long table", "/clickhouse/databases/default/tables/" + tooLong, "invalid table name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeClickHouseConn{}
			router := newClickHouseTestRouter(conn)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if w.Code != http.StatusBadRequest {
				t.Fatalf("GET %s = %d, want %d", tt.path, w.Code, http.StatusBadRequest)
			}
			var body ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error != tt.wantError {
				t.Errorf("error = %q, want %q", body.Error, tt.wantError)
			}
			if len(conn.queries) != 0 {
				t.Errorf("got %d queries, want none", len(conn.queries))
			}
		})
	}
}
