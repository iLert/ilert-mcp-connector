package main

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// identifierPattern restricts database and table names taken from request paths.
// It covers MySQL unquoted identifiers ([A-Za-z0-9_$]), hyphenated names commonly
// used for databases, and dots used by ClickHouse internal tables (e.g. ".inner.mv").
// Backticks, quotes, whitespace and other control characters are rejected.
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9_$.-]{1,128}$`)

func validIdentifier(s string) bool {
	return identifierPattern.MatchString(s)
}

// validateDatabaseParam reads the :database path param and writes a 400 response if it is invalid.
func validateDatabaseParam(c *gin.Context) (string, bool) {
	database := c.Param("database")
	if !validIdentifier(database) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid database name"})
		return "", false
	}
	return database, true
}

// validateTableParam reads the :table path param and writes a 400 response if it is invalid.
func validateTableParam(c *gin.Context) (string, bool) {
	table := c.Param("table")
	if !validIdentifier(table) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid table name"})
		return "", false
	}
	return table, true
}
