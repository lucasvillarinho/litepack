package cache

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/database/drivers"
	"github.com/lucasvillarinho/litepack/schedule"
)

// cache is a simple key-value store backed by an SQLite database.
type cache struct {
	scheduler    schedule.Scheduler
	engine       drivers.Driver
	drive        drivers.DriverType
	timezone     *time.Location
	dsn          string
	syncInterval schedule.Interval
	cacheSize    int
}

type Cache interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Del(key string) error
	Close() error
	Destroy() error
}

// NewCache creates a new cache instance with the given name and applies any provided options.
// The cache is backed by an SQLite database.
// The path is used to create a database file with the format "<name>_lpack_cache.db".
// The cache is automatically created if it does not exist.
//
// Parameters:
//   - path: the path of the cache database
//   - opts: the cache options
//
// Configuration defaults:
//   - syncInterval: 1 second
//   - timezone: UTC
//
// Configuration options:
//   - WithSyncInterval: sets a custom sync interval for the cache.
//   - WithTimezone: sets a custom timezone for the cache.
//
// Returns:
//   - *cache: the cache instance
//   - error: an error if the operation failed
func NewCache(path string, opts ...Option) (Cache, error) {
	c := &cache{
		dsn:          fmt.Sprintf("%s_lpack_cache.db", path),
		syncInterval: schedule.EveryMinute,
		timezone:     time.UTC,
		cacheSize:    128 * 1024 * 1024, // 128 MB
		drive:        drivers.DriverMattn,
	}

	for _, opt := range opts {
		opt(c)
	}

	err := setupTable(c)
	if err != nil {
		return nil, fmt.Errorf("error setting up cache table: %w", err)
	}

	err = startSyncClearByTTL(c)
	if err != nil {
		return nil, fmt.Errorf("error setting up sync clear by TTL: %w", err)
	}

	return c, nil
}

// Set sets a key-value pair in the cache with the given TTL.
// If the key already exists, it is updated with the new value and TTL.
// The key-value pair is automatically removed from the cache after the TTL expires.
//
// Parameters:
//   - key: the cache key
//   - value: the cache value
//   - ttl: the time-to-live for the cache entry
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) Set(key string, value []byte, ttl time.Duration) error {
	_, err := ch.engine.Execute(
		`INSERT OR REPLACE INTO cache (key, value, expires_at) 
		 VALUES (?, ?, ?);`,
		key,
		value,
		time.Now().Add(ttl).In(ch.timezone),
	)
	return err
}

// Get retrieves a value from the cache by key.
//
// Parameters:
//   - key: the cache key
//
// Returns:
//   - []byte: the cache value
//   - error: an error if the operation failed
func (ch *cache) Get(key string) ([]byte, error) {
	var value []byte

	// Query only non-expired records
	err := ch.engine.
		QueryRow(
			`SELECT value FROM cache WHERE key = ? AND expires_at > ?;`,
			key,
			time.Now().In(ch.timezone),
		).
		Scan(&value)
	if err != nil {
		// Return nil if the key does not exist
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return value, nil
}

// Del deletes a key-value pair from the cache.
// If the key does not exist, the operation is a no-op.
//
// Parameters:
//   - key: the cache key
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) Del(key string) error {
	_, err := ch.engine.Execute(`DELETE FROM cache WHERE key = ?;`, key)
	if err != nil {
		fmt.Println("error deleting key", err)
	}

	return err
}

// clearExpiredItems Deletes all cache entries that have expired.
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) clearExpiredItems() error {
	_, err := ch.engine.Execute(`
		DELETE FROM cache WHERE expires_at <= ?;
	`, time.Now().In(ch.timezone))
	if err != nil {
		return fmt.Errorf("clearing expired items: %w", err)
	}

	return nil
}

// Close closes the cache database connection.
func (ch *cache) Close() error {
	return ch.engine.Close()
}

// Destroy deletes the cache database file and closes the database connection.
//
// WARNING: THIS OPERATION IS IRREVERSIBLE.
func (ch *cache) Destroy() error {
	err := ch.Close()
	if err != nil {
		return err
	}
	return database.DeleteDatabase(ch.dsn)
}
