package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var authToken string

func initAuth() {
	// Check if auth token is provided via environment variable
	envToken := os.Getenv("AUTH_TOKEN")
	if envToken != "" {
		authToken = envToken
		log.Info().Msg("Using authentication token from AUTH_TOKEN environment variable")
		return
	}

	// Generate a new random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate authentication token")
	}
	authToken = hex.EncodeToString(tokenBytes)
	log.Info().Str("token", authToken).Msg("Generated new authentication token")
}

// AuthMiddleware validates the Authorization header token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn().
				Str("ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Unauthorized request: missing Authorization header")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Unauthorized: missing Authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" or just "<token>"
		token := strings.TrimSpace(authHeader)
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
			token = strings.TrimSpace(token)
		}

		if token != authToken {
			log.Warn().
				Str("ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Unauthorized request: invalid token")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Unauthorized: invalid token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
