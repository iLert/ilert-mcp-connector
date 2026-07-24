package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildKeyValues(t *testing.T) {
	large := strings.Repeat("x", maxValueBytes+50)

	// MGET returns nil for both missing and non-string keys; `types` (populated by
	// the handler only for nil entries) disambiguates them.
	keys := []string{"missing", "small", "large", "nonstring", "binary"}
	vals := []interface{}{nil, "hello", large, nil, "\xff\xfe\x01data"}
	types := []string{"none", "", "", "list", ""}

	out := buildKeyValues(keys, vals, types)

	if len(out) != len(keys) {
		t.Fatalf("expected %d entries, got %d", len(keys), len(out))
	}

	// missing key -> status MISSING, nil value
	if out[0].Status != KeyValueStatusMissing || out[0].Value != nil {
		t.Errorf("missing key: expected status=MISSING value=nil, got %+v", out[0])
	}

	// small string -> status OK, untruncated, size == byte length
	if out[1].Status != KeyValueStatusOK || out[1].Truncated || out[1].Size != len("hello") || out[1].Value == nil || *out[1].Value != "hello" {
		t.Errorf("small key: unexpected result %+v", out[1])
	}

	// large string -> status OK, truncated to maxValueBytes, size == original length
	if out[2].Status != KeyValueStatusOK || !out[2].Truncated || out[2].Value == nil {
		t.Errorf("large key: expected status=OK truncated=true with value, got %+v", out[2])
	}
	if out[2].Size != len(large) {
		t.Errorf("large key: expected size=%d (original), got %d", len(large), out[2].Size)
	}
	if len(*out[2].Value) > maxValueBytes {
		t.Errorf("large key: expected value truncated to at most %d bytes, got %d", maxValueBytes, len(*out[2].Value))
	}

	// non-string key -> status NON_STRING, nil value
	if out[3].Status != KeyValueStatusNonString || out[3].Value != nil {
		t.Errorf("nonstring key: expected status=NON_STRING value=nil, got %+v", out[3])
	}

	// non-UTF-8 value -> status BINARY, nil value, size == byte length
	if out[4].Status != KeyValueStatusBinary || out[4].Value != nil || out[4].Size != len("\xff\xfe\x01data") {
		t.Errorf("binary key: expected status=BINARY value=nil, got %+v", out[4])
	}
}

func newValuesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewRedisHandler(RedisConfig{EnableSensitiveTools: true})
	r.GET("/redis/databases/:database/values", h.GetKeyValues)
	return r
}

func TestGetKeyValuesSensitiveDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewRedisHandler(RedisConfig{EnableSensitiveTools: false})
	r.GET("/redis/databases/:database/values", h.GetKeyValues)

	req := httptest.NewRequest(http.MethodGet, `/redis/databases/0/values?keys=`+url.QueryEscape(`["a"]`), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestGetKeyValuesValidation(t *testing.T) {
	tests := []struct {
		name       string
		database   string
		keysParam  string // raw (un-encoded) value of the keys query param; "" means omit
		wantStatus int
	}{
		{"invalid database", "abc", `["a"]`, http.StatusBadRequest},
		{"malformed keys", "0", `not-json`, http.StatusBadRequest},
		{"empty keys array", "0", `[]`, http.StatusBadRequest},
		{"missing keys param", "0", "", http.StatusBadRequest},
		{"too many keys", "0", tooManyKeysJSON(21), http.StatusBadRequest},
	}

	router := newValuesTestRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := "/redis/databases/" + tt.database + "/values"
			if tt.keysParam != "" {
				target += "?keys=" + url.QueryEscape(tt.keysParam)
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func tooManyKeysJSON(n int) string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "k" + strconv.Itoa(i)
	}
	b, _ := json.Marshal(keys)
	return string(b)
}
