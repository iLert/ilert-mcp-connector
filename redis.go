package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	maxKeys       = 20
	maxValueBytes = 1024
)

type RedisHandler struct {
	config RedisConfig
}

func NewRedisHandler(config RedisConfig) *RedisHandler {
	return &RedisHandler{
		config: config,
	}
}

func (h *RedisHandler) getClient(db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", h.config.Host, h.config.Port),
		Password: h.config.Password,
		DB:       db,
	})
}

func checkRedisConnection(config RedisConfig) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       0,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return client.Ping(ctx).Err()
}

func (h *RedisHandler) ScanKeys(c *gin.Context) {
	databaseStr := c.Param("database")
	database, err := strconv.Atoi(databaseStr)
	if err != nil {
		log.Error().Err(err).Str("database", databaseStr).Msg("Invalid database number")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid database number"})
		return
	}

	client := h.getClient(database)
	defer client.Close()

	pattern := c.DefaultQuery("pattern", "*")
	cursorStr := c.DefaultQuery("cursor", "0")
	cursor, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		log.Error().Err(err).Str("cursor", cursorStr).Msg("Invalid cursor value")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid cursor value"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batchSize := int64(250)

	keys, nextCursor, err := client.Scan(ctx, cursor, pattern, batchSize).Result()
	recordRedisOperation("scan_keys", err)
	if err != nil {
		log.Error().Err(err).Str("pattern", pattern).Int("database", database).Msg("Failed to scan Redis keys")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, KeysResponse{
		Keys:    keys,
		Count:   len(keys),
		Pattern: pattern,
		Cursor:  nextCursor,
		HasMore: nextCursor != 0,
	})
}

func (h *RedisHandler) GetKeyInfo(c *gin.Context) {
	databaseStr := c.Param("database")
	database, err := strconv.Atoi(databaseStr)
	if err != nil {
		log.Error().Err(err).Str("database", databaseStr).Msg("Invalid database number")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid database number"})
		return
	}

	client := h.getClient(database)
	defer client.Close()

	key := c.Param("key")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keyType, err := client.Type(ctx, key).Result()
	recordRedisOperation("get_key_info", err)
	if err != nil {
		log.Error().Err(err).Str("key", key).Int("database", database).Msg("Failed to get Redis key type")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Str("key", key).Int("database", database).Msg("Failed to get Redis key TTL")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	info := KeyInfo{
		Key:  key,
		Type: keyType,
		TTL:  int64(ttl.Seconds()),
	}

	if keyType == "list" {
		length, err := client.LLen(ctx, key).Result()
		if err == nil {
			info.Size = length
		}
	} else if keyType == "set" {
		length, err := client.SCard(ctx, key).Result()
		if err == nil {
			info.Size = length
		}
	} else if keyType == "zset" {
		length, err := client.ZCard(ctx, key).Result()
		if err == nil {
			info.Size = length
		}
	} else if keyType == "hash" {
		length, err := client.HLen(ctx, key).Result()
		if err == nil {
			info.Size = length
		}
	} else if keyType == "stream" {
		length, err := client.XLen(ctx, key).Result()
		if err == nil {
			info.Size = length
		}
	}

	memory, err := client.MemoryUsage(ctx, key).Result()
	if err == nil {
		info.Size = memory
	}

	encoding, err := client.ObjectEncoding(ctx, key).Result()
	if err == nil {
		info.Encoding = encoding
	}

	c.JSON(http.StatusOK, KeyInfoResponse{Info: info})
}

func (h *RedisHandler) GetKeyValues(c *gin.Context) {
	if !h.config.EnableSensitiveTools {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "This endpoint is disabled in the settings",
		})
		return
	}

	databaseStr := c.Param("database")
	database, err := strconv.Atoi(databaseStr)
	if err != nil {
		log.Error().Err(err).Str("database", databaseStr).Msg("Invalid database number")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid database number"})
		return
	}

	raw := c.Query("keys")
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		log.Error().Err(err).Str("keys", raw).Msg("Invalid keys parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid keys parameter"})
		return
	}
	if len(keys) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid keys parameter"})
		return
	}
	if len(keys) > maxKeys {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Too many keys (max 20)"})
		return
	}

	client := h.getClient(database)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vals, err := client.MGet(ctx, keys...).Result()
	recordRedisOperation("get_key_values", err)
	if err != nil {
		log.Error().Err(err).Int("database", database).Msg("Failed to get Redis key values")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// MGET returns nil for both missing keys and keys holding a non-string type.
	// Look up TYPE for the nil entries so we can distinguish "missing" from
	// "exists but non-string" (reported via skippedNonString).
	types := make([]string, len(keys))
	for i := range keys {
		if i < len(vals) && vals[i] == nil {
			if t, terr := client.Type(ctx, keys[i]).Result(); terr == nil {
				types[i] = t
			}
		}
	}

	out := buildKeyValues(keys, vals, types)
	c.JSON(http.StatusOK, ValuesResponse{Values: out})
}

func buildKeyValues(keys []string, vals []interface{}, types []string) []KeyValue {
	out := make([]KeyValue, len(keys))
	for i, key := range keys {
		kv := KeyValue{Key: key, Status: KeyValueStatusMissing}
		if i < len(vals) {
			if s, ok := vals[i].(string); ok {
				kv.Size = len(s)
				if !utf8.ValidString(s) {
					kv.Status = KeyValueStatusBinary
					out[i] = kv
					continue
				}
				if kv.Size > maxValueBytes {
					s = strings.ToValidUTF8(s[:maxValueBytes], "")
					kv.Truncated = true
				}
				kv.Value = &s
				kv.Status = KeyValueStatusOK
				out[i] = kv
				continue
			}
		}
		// value is nil: distinguish a non-string key from a missing key.
		if i < len(types) && types[i] != "" && types[i] != "none" {
			kv.Status = KeyValueStatusNonString
		}
		out[i] = kv
	}
	return out
}

func (h *RedisHandler) GetMetrics(c *gin.Context) {
	client := h.getClient(0)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := RedisMetrics{}

	infoStr, err := client.Info(ctx).Result()
	recordRedisOperation("get_metrics", err)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get Redis INFO")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	lines := splitString(infoStr, "\n")
	currentSection := ""

	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "" {
			continue
		}

		if len(trimmed) > 0 && trimmed[0] == '#' {
			sectionName := trimSpace(trimmed[1:])
			if sectionName != "" {
				currentSection = sectionName
			}
			continue
		}

		parts := splitString(trimmed, ":")
		if len(parts) >= 2 {
			key := trimSpace(parts[0])
			value := trimSpace(parts[1])
			if key != "" {
				if currentSection == "Server" {
					if metrics.Server == nil {
						metrics.Server = make(map[string]string)
					}
					metrics.Server[key] = value
				} else if currentSection == "Memory" {
					if metrics.Memory == nil {
						metrics.Memory = make(map[string]string)
					}
					metrics.Memory[key] = value
				} else if currentSection == "Stats" {
					if metrics.Stats == nil {
						metrics.Stats = make(map[string]string)
					}
					metrics.Stats[key] = value
				} else if currentSection == "Clients" {
					if metrics.Clients == nil {
						metrics.Clients = make(map[string]string)
					}
					metrics.Clients[key] = value
				} else if currentSection == "Persistence" {
					if metrics.Persistence == nil {
						metrics.Persistence = make(map[string]string)
					}
					metrics.Persistence[key] = value
				} else if currentSection == "Replication" {
					if metrics.Replication == nil {
						metrics.Replication = make(map[string]string)
					}
					metrics.Replication[key] = value
				} else if currentSection == "CPU" {
					if metrics.CPU == nil {
						metrics.CPU = make(map[string]string)
					}
					metrics.CPU[key] = value
				} else if currentSection == "Keyspace" {
					if metrics.Keyspace == nil {
						metrics.Keyspace = make(map[string]string)
					}
					metrics.Keyspace[key] = value
				}
			}
		}
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *RedisHandler) GetInfo(c *gin.Context) {
	client := h.getClient(0)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	infoStr, err := client.Info(ctx).Result()
	recordRedisOperation("get_info", err)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get Redis INFO")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	infoMap := make(map[string]string)
	lines := splitString(infoStr, "\n")
	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		parts := splitString(trimmed, ":")
		if len(parts) >= 2 {
			key := trimSpace(parts[0])
			value := trimSpace(parts[1])
			if key != "" {
				infoMap[key] = value
			}
		}
	}

	c.JSON(http.StatusOK, RedisInfoResponse{Info: infoMap})
}

func (h *RedisHandler) GetDatabases(c *gin.Context) {
	client := h.getClient(0)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configMap, err := client.ConfigGet(ctx, "databases").Result()
	recordRedisOperation("get_databases", err)
	if err != nil {
		log.Debug().Err(err).Msg("CONFIG command not available, using default database 0")
		c.JSON(http.StatusOK, RedisDatabasesResponse{Databases: []int{0}})
		return
	}

	databases := []int{}
	if dbCountStr, ok := configMap["databases"]; ok {
		if dbCount, err := strconv.Atoi(dbCountStr); err == nil {
			for i := 0; i < dbCount; i++ {
				databases = append(databases, i)
			}
		}
	}

	if len(databases) == 0 {
		databases = []int{0}
	}

	c.JSON(http.StatusOK, RedisDatabasesResponse{Databases: databases})
}
