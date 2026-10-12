package main

import (
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// MySQL has no placeholder for identifiers (SHOW TABLES FROM, DESCRIBE), so database and table names
// taken from request paths are validated and quoted instead of bound. The rules follow the MySQL
// reference manual for quoted identifiers (https://dev.mysql.com/doc/refman/8.4/en/identifiers.html,
// identifier-length.html): any BMP character except U+0000 (no supplementary characters), at most
// 64 characters, and database / table names must not end with a space.
//
// ClickHouse queries bind database and table names as values, so only their length is capped: ClickHouse
// stores each database and table under a file named after it, and file names are limited to 255 bytes.
const clickHouseMaxNameBytes = 255

var mysqlIdentifierPattern = regexp.MustCompile(`^[\x{0001}-\x{FFFF}]{0,63}[\x{0001}-\x{001F}\x{0021}-\x{FFFF}]$`)

func validMySQLIdentifier(s string) bool {
	// invalid UTF-8 bytes would otherwise match as U+FFFD
	return utf8.ValidString(s) && mysqlIdentifierPattern.MatchString(s)
}

// quoteMySQLIdentifier wraps a validated name in backticks, doubling any backtick inside it.
func quoteMySQLIdentifier(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

// validateMySQLDatabaseParam reads the :database path param and writes a 400 response if it is invalid.
func validateMySQLDatabaseParam(c *gin.Context) (string, bool) {
	database := c.Param("database")
	if !validMySQLIdentifier(database) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid database name"})
		return "", false
	}
	return database, true
}

// validateMySQLTableParam reads the :table path param and writes a 400 response if it is invalid.
func validateMySQLTableParam(c *gin.Context) (string, bool) {
	table := c.Param("table")
	if !validMySQLIdentifier(table) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid table name"})
		return "", false
	}
	return table, true
}

func validClickHouseName(s string) bool {
	return len(s) > 0 && len(s) <= clickHouseMaxNameBytes
}

// validateClickHouseDatabaseParam reads the :database path param and writes a 400 response if it is invalid.
func validateClickHouseDatabaseParam(c *gin.Context) (string, bool) {
	database := c.Param("database")
	if !validClickHouseName(database) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid database name"})
		return "", false
	}
	return database, true
}

// validateClickHouseTableParam reads the :table path param and writes a 400 response if it is invalid.
func validateClickHouseTableParam(c *gin.Context) (string, bool) {
	table := c.Param("table")
	if !validClickHouseName(table) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid table name"})
		return "", false
	}
	return table, true
}
