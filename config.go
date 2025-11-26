package main

import (
	"os"
)

type Config struct {
	Kafka      KafkaConfig
	MySQL      MySQLConfig
	ClickHouse ClickHouseConfig
}

type KafkaConfig struct {
	Enabled  bool
	Brokers  []string
	ClientID string
}

type MySQLConfig struct {
	Enabled  bool
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type ClickHouseConfig struct {
	Enabled  bool
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func LoadConfig() *Config {
	cfg := &Config{
		Kafka: KafkaConfig{
			Enabled:  getEnvBool("KAFKA_ENABLED", false),
			Brokers:  getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ClientID: getEnv("KAFKA_CLIENT_ID", "ilert-mcp-connector"),
		},
		MySQL: MySQLConfig{
			Enabled:  getEnvBool("MYSQL_ENABLED", false),
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", ""),
			Database: getEnv("MYSQL_DATABASE", ""),
		},
		ClickHouse: ClickHouseConfig{
			Enabled:  getEnvBool("CLICKHOUSE_ENABLED", false),
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnv("CLICKHOUSE_PORT", "9000"),
			User:     getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
			Database: getEnv("CLICKHOUSE_DATABASE", "default"),
		},
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		var result []string
		for _, v := range splitString(value, ",") {
			if trimmed := trimSpace(v); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// Helper functions to avoid importing strings package
func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
