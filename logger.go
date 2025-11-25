package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initLogger() {
	// Set log level from environment or default to info
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output format
	if os.Getenv("LOG_FORMAT") == "json" {
		// JSON format for production
		log.Logger = zerolog.New(os.Stdout).With().
			Timestamp().
			Logger()
	} else {
		// Pretty console format for development
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}

// AccessLogMiddleware returns a gin middleware for access logging
func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		event := log.Info()
		if statusCode >= 400 && statusCode < 500 {
			event = log.Warn()
		} else if statusCode >= 500 {
			event = log.Error()
		}

		event.
			Str("method", method).
			Str("path", path).
			Str("query", raw).
			Int("status", statusCode).
			Str("ip", clientIP).
			Dur("latency", duration).
			Str("user_agent", c.Request.UserAgent())

		if errorMessage != "" {
			event.Str("error", errorMessage)
		}

		event.Msg("HTTP request")
	}
}
