package database

import (
	"fmt"
	"os"
	"strings"
)

// DeleteDatabase deletes the given database file
//
// Parameters:
//   - db: the database connection
//
// Returns:
//   - error: an error if the operation failed
func DeleteDatabase(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete database file: %w", err)
	}

	return nil
}

// IsDatabaseFullError checks if the given error is a database full error
func IsDatabaseFullError(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "database or disk is full") {
		return true
	}

	return false
}
