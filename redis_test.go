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
	keys := []string{"missing", "small", "large", "nonstring"}
	vals := []interface{}{nil, "hello", large, nil}
	types := []string{"none", "", "", "list"}

	out := buildKeyValues(keys, vals, types)

	if len(out) != len(keys) {
		t.Fatalf("expected %d entries, got %d", len(keys), len(out))
	}

	// missing key -> not found, not skipped, nil value
	if out[0].Found || out[0].SkippedNonString || out[0].Value != nil {
		t.Errorf("missing key: expected found=false skipped=false value=nil, got %+v", out[0])
	}

	// small string -> found, untruncated, size == byte length
	if !out[1].Found || out[1].Truncated || out[1].Size != len("hello") || *out[1].Value != "hello" {
		t.Errorf("small key: unexpected result %+v", out[1])
	}

	// large string -> found, truncated to maxValueBytes, size == original length
	if !out[2].Found || !out[2].Truncated {
		t.Errorf("large key: expected found=true truncated=true, got %+v", out[2])
	}
	if out[2].Size != len(large) {
		t.Errorf("large key: expected size=%d (original), got %d", len(large), out[2].Size)
	}
	if len(*out[2].Value) != maxValueBytes {
		t.Errorf("large key: expected value truncated to %d bytes, got %d", maxValueBytes, len(*out[2].Value))
	}

	// non-string key -> not found, skippedNonString=true, nil value
	if out[3].Found || !out[3].SkippedNonString || out[3].Value != nil {
		t.Errorf("nonstring key: expected found=false skipped=true value=nil, got %+v", out[3])
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
