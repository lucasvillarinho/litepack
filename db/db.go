package db

import (
	"database/sql"
	"fmt"
	"os"
)

// DeleteDatabase deletes the given database file
// and closes the database connection.
//
// Parameters:
//   - db: the database connection
//
// Returns:
//   - error: an error if the operation failed
func DeleteDatabase(db *sql.DB, path string) error {
	if err := db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete database file: %w", err)
	}

	return nil
}
