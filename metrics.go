package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	kafkaOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_operations_total",
			Help: "Total number of Kafka operations",
		},
		[]string{"operation", "status"},
	)

	mysqlOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mysql_operations_total",
			Help: "Total number of MySQL operations",
		},
		[]string{"operation", "status"},
	)

	mysqlConnectionsOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_connections_open",
			Help: "Number of open MySQL connections",
		},
		[]string{"state"},
	)

	clickhouseOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clickhouse_operations_total",
			Help: "Total number of ClickHouse operations",
		},
		[]string{"operation", "status"},
	)

	redisOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)
)

func initMetrics() {
	prometheus.MustRegister(kafkaOperationsTotal)
	prometheus.MustRegister(mysqlOperationsTotal)
	prometheus.MustRegister(mysqlConnectionsOpen)
	prometheus.MustRegister(clickhouseOperationsTotal)
	prometheus.MustRegister(redisOperationsTotal)
}

func metricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func recordKafkaOperation(operation string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	kafkaOperationsTotal.WithLabelValues(operation, status).Inc()
}

func recordMySQLOperation(operation string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	mysqlOperationsTotal.WithLabelValues(operation, status).Inc()
}

func recordClickHouseOperation(operation string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	clickhouseOperationsTotal.WithLabelValues(operation, status).Inc()
}

func recordRedisOperation(operation string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	redisOperationsTotal.WithLabelValues(operation, status).Inc()
}

func updateMySQLConnectionMetrics(db *sql.DB) {
	if db == nil {
		return
	}
	stats := db.Stats()
	mysqlConnectionsOpen.WithLabelValues("open").Set(float64(stats.OpenConnections))
	mysqlConnectionsOpen.WithLabelValues("idle").Set(float64(stats.Idle))
	mysqlConnectionsOpen.WithLabelValues("in_use").Set(float64(stats.InUse))
}
