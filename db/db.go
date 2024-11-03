package db

import (
	"database/sql"
	"fmt"
	"os"
)

// DeleteDatabase deletes the given database file
//
// Parameters:
//   - db: the database connection
//
// Returns:
//   - error: an error if the operation failed
func DeleteDatabase(db *sql.DB, path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete database file: %w", err)
	}

	return nil
}
