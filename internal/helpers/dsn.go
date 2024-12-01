package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateDSN creates a DSN string for an SQLite database.
//
// If the path is empty, the current directory is used
// to create the database file.
//
// Parameters:
//   - path: the path to the database file
//   - db: the database file name
//
// Returns:
//   - dsn: the DSN string
//   - error: an error if the operation failed
func CreateDSN(path, db string) (string, error) {
	var dsn string

	if path == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("falha ao obter o diretório atual: %w", err)
		}

		return filepath.Join(currentDir, db), nil
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", fmt.Errorf("falha ao criar diretórios: %w", err)
	}
	dsn = filepath.Join(path, db)

	return dsn, nil
}
