package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func main() {
	// Set Gin mode to release if not set by user
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	initLogger()

	// Initialize authentication
	initAuth()

	// Load configuration
	cfg := LoadConfig()

	log.Info().
		Str("version", Version).
		Str("commit", Commit).
		Msg("Starting ilert-mcp-connector")

	// Initialize router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(AccessLogMiddleware())

	// Public endpoints (no auth required)
	router.GET("/health", healthHandler)
	router.GET("/version", versionHandler)

	// Protected routes group (requires authentication)
	protected := router.Group("/")
	protected.Use(AuthMiddleware())

	// Protected readiness endpoint (contains sensitive connection info)
	protected.GET("/ready", readyHandler(cfg))

	// Initialize tool handlers
	if cfg.Kafka.Enabled {
		kafkaHandler := NewKafkaHandler(cfg.Kafka)
		protected.GET("/kafka/topics", kafkaHandler.ListTopics)
		protected.GET("/kafka/topics/:topic", kafkaHandler.DescribeTopic)
		protected.GET("/kafka/consumers", kafkaHandler.ListConsumers)
		protected.GET("/kafka/consumers/:group", kafkaHandler.DescribeConsumer)
		protected.GET("/kafka/consumers/:group/lag", kafkaHandler.GetConsumerLag)
	}

	if cfg.MySQL.Enabled {
		mysqlHandler := NewMySQLHandler(cfg.MySQL)
		protected.GET("/mysql/databases", mysqlHandler.GetDatabases)
		protected.GET("/mysql/databases/:database/tables", mysqlHandler.GetTables)
		protected.GET("/mysql/databases/:database/tables/:table", mysqlHandler.DescribeTable)
		protected.GET("/mysql/metrics", mysqlHandler.GetMetrics)
	}

	if cfg.ClickHouse.Enabled {
		clickhouseHandler := NewClickHouseHandler(cfg.ClickHouse)
		protected.GET("/clickhouse/databases", clickhouseHandler.GetDatabases)
		protected.GET("/clickhouse/databases/:database/tables", clickhouseHandler.GetTables)
		protected.GET("/clickhouse/databases/:database/tables/:table", clickhouseHandler.DescribeTable)
		protected.GET("/clickhouse/metrics", clickhouseHandler.GetMetrics)
	}

	// Create HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8383"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("addr", srv.Addr).
			Str("version", Version).
			Str("commit", Commit).
			Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		os.Exit(1)
	}

	log.Info().Msg("Server exited")
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Version: Version,
		Commit:  Commit,
	})
}

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "healthy"})
}

type ReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func readyHandler(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ready := true
		checks := make(map[string]string)

		if cfg.Kafka.Enabled {
			// Check Kafka connection
			if err := checkKafkaConnection(cfg.Kafka); err != nil {
				ready = false
				checks["kafka"] = err.Error()
			} else {
				checks["kafka"] = "ok"
			}
		}

		if cfg.MySQL.Enabled {
			// Check MySQL connection
			if err := checkMySQLConnection(cfg.MySQL); err != nil {
				ready = false
				checks["mysql"] = err.Error()
			} else {
				checks["mysql"] = "ok"
			}
		}

		if cfg.ClickHouse.Enabled {
			// Check ClickHouse connection
			if err := checkClickHouseConnection(cfg.ClickHouse); err != nil {
				ready = false
				checks["clickhouse"] = err.Error()
			} else {
				checks["clickhouse"] = "ok"
			}
		}

		if ready {
			c.JSON(http.StatusOK, ReadyResponse{Status: "ready", Checks: checks})
		} else {
			c.JSON(http.StatusServiceUnavailable, ReadyResponse{Status: "not ready", Checks: checks})
		}
	}
}
