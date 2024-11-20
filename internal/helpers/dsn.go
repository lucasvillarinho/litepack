package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateDSN(path string) (string, error) {
	var dsn string

	if path == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("falha ao obter o diretório atual: %w", err)
		}

		return filepath.Join(currentDir, "lpack_cache.db"), nil
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", fmt.Errorf("falha ao criar diretórios: %w", err)
	}
	dsn = filepath.Join(path, "lpack_cache.db")

	return dsn, nil
}
