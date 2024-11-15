package cache

import (
	"fmt"

	"github.com/lucasvillarinho/litepack/database/drivers"
)

// SetupTable creates the cache table with custom configuration.

// Returns:
//   - error: an error if the operation failed
func setupTable(ch *cache) error {
	if err := setDriver(ch, drivers.NewDriverFactory()); err != nil {
		return err
	}

	if err := createCacheTable(ch); err != nil {
		return err
	}

	if err := createIndex(ch); err != nil {
		return err
	}

	if err := setWalMode(ch); err != nil {
		return err
	}

	if err := setCacheSize(ch); err != nil {
		return err
	}

	if err := setSynchronousMode(ch); err != nil {
		return err
	}

	return nil
}

// setWalMode enables Write-Ahead Logging (WAL) mode for the database.
// WAL mode allows for concurrent reads and writes to the database.
// WAL mode is recommended for high-traffic applications.
//
// Parameters:
//   - ch: the cache handle
//
// Returns:
//   - error: an error if the operation failed
func setWalMode(ch *cache) error {
	_, err := ch.engine.Execute("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("enabling WAL mode: %w", err)
	}
	return nil
}

// setSynchronousMode sets the synchronous mode for the database.
// Synchronous mode determines how often the database writes to disk.
func setSynchronousMode(ch *cache) error {
	_, err := ch.engine.Execute("PRAGMA synchronous = NORMAL;")
	if err != nil {
		return fmt.Errorf("setting synchronous mode: %w", err)
	}
	return nil
}

// createIndex creates an index on the cache table for the key column.
//
// Parameters:
//   - ch: the cache handle
//
// Returns:
//   - error: an error if the operation failed
func createIndex(ch *cache) error {
	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_key ON cache (key);`
	_, err := ch.engine.Execute(createIndexSQL)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	return nil
}

// setCacheSize sets the cache size for the database.
// The cache size is set in pages, with each page being 4096 bytes.
// The default cache size is 128 MB.
//
// This cache is used by SQLite to store data pages in memory,
// minimizing the need for direct disk access.
//
// Parameters:
//
//   - ch: the cache handle
//
// Returns:
//
//   - error: an error if the operation failed
func setCacheSize(ch *cache) error {
	pages := ch.cacheSize / 4096

	query := fmt.Sprintf("PRAGMA cache_size = %d;", pages)

	_, err := ch.engine.Execute(query)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	return nil
}

// createCacheTable creates the cache table if it does not exist.
//
// The table has the following schema:
//
//   - key: TEXT PRIMARY KEY
//   - value: BLOB
//   - expires_at: TIMESTAMP
//   - created_at: TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//   - last_accessed_at: TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//
// Parameters:
//   - ch: the cache handle
//
// Returns:
//   - error: an error if the operation failed
func createCacheTable(ch *cache) error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS cache (
        key TEXT PRIMARY KEY,
        value BLOB,
        expires_at TIMESTAMP,
        last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	_, err := ch.engine.Execute(createTableSQL)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}
	return nil
}

// setDriver sets the driver for the cache.
// The driver is used to interact with the SQLite database.
//
// Configuration defaults:
//   - driver: mattn
//
// Parameters:
//
//   - ch: the cache handle
//   - driverFactory: the driver factory
//
// Returns:
//   - error: an error if the operation failed
func setDriver(ch *cache, driverFactory drivers.DriverFactory) error {
	engine, err := driverFactory.GetDriver(ch.drive, ch.dsn)
	if err != nil {
		return fmt.Errorf("error getting driver: %w", err)
	}
	ch.engine = engine

	return nil
}
