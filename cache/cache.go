package cache

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/schedule"
)

// cache is a simple key-value store backed by an SQLite database.
type cache struct {
	scheduler    schedule.Scheduler
	db           *sql.DB
	timezone     *time.Location
	url          string
	syncInterval schedule.ScheduleTime
}

type Cache interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Del(key string) error
	Close() error
	Destroy() error
}

// CacheOption is a function that configures a cache instance.
type Option func(*cache)

// WithClearInterval sets a custom sync interval for the cache.
func WithClearInterval(interval schedule.ScheduleTime) Option {
	return func(c *cache) {
		c.syncInterval = interval
	}
}

// WithTimezone sets a custom timezone for the cache.
func WithTimezone(location *time.Location) Option {
	return func(c *cache) {
		c.timezone = location
	}
}

// NewCache creates a new cache instance with the given name and applies any provided options.
// The cache is backed by an SQLite database.
// The name is used to create a database file with the format "<name>_pack_cache.db".
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
func NewCache(url string, opts ...Option) (Cache, error) {
	c := &cache{
		url:          fmt.Sprintf("%s_cache.db", url),
		syncInterval: schedule.EveryMinute,
		timezone:     time.UTC,
	}

	for _, opt := range opts {
		opt(c)
	}

	db, err := sql.Open("sqlite3", c.url)
	if err != nil {
		return nil, err
	}
	c.db = db

	err = createCacheTable(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache table: %w", err)
	}

	// Create a new scheduler with the cache timezone
	c.scheduler = schedule.NewScheduler(c.timezone)

	// // Start a background goroutine to clear expired cache entries
	go c.scheduler.Task(schedule.EveryMinute, c.clearExpiredItems)

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
	_, err := ch.db.Exec(
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
	var expiresAt time.Time

	err := ch.db.
		QueryRow(`SELECT value, expires_at FROM cache WHERE key = ?;`, key).
		Scan(&value, &expiresAt)
	if err != nil {
		// Return nil if the key does not exist
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	// Check if the entry has expired
	// If the entry has expired, remove it from the cache
	if time.Now().In(ch.timezone).After(expiresAt) {
		if delErr := ch.Del(key); delErr != nil {
			return nil, delErr
		}
		// Return nil if expired
		return nil, nil
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
	_, err := ch.db.Exec(`DELETE FROM cache WHERE key = ?;`, key)
	return err
}

// clearExpiredItems Deletes all cache entries that have expired.
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) clearExpiredItems() error {
	_, err := ch.db.Exec(`
		DELETE FROM cache WHERE expires_at <= ?;
	`, time.Now().In(ch.timezone))
	if err != nil {
		return fmt.Errorf("failed to clear expired cache entries: %w", err)
	}

	return nil
}

// Close closes the cache database connection.
func (ch *cache) Close() error {
	return ch.db.Close()
}

// Destroy deletes the cache database file and closes the database connection.
//
// WARNING: THIS OPERATION IS IRREVERSIBLE.
func (ch *cache) Destroy() error {
	err := ch.Close()
	if err != nil {
		return err
	}
	return database.DeleteDatabase(ch.url)
}

// createCacheTable creates the cache table if it does not exist.
//
// The table has the following schema:
//
//   - key: TEXT PRIMARY KEY
//   - value: BLOB
//   - expires_at: TIMESTAMP
//   - created_at: TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//
// Parameters:
//   - db: the database handle
//
// Returns:
//   - error: an error if the operation failed
func createCacheTable(db *sql.DB) error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS cache (
        key TEXT PRIMARY KEY,
        value BLOB,
        expires_at TIMESTAMP,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}
	return nil
}

// setIndex creates an index on the cache table for the key column.
//
// Parameters:
//   - db: the database handle
//
// Returns:
//   - error: an error if the operation failed
func setIndex(db *sql.DB) error {
	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_key ON cache (key);`
	_, err := db.Exec(createIndexSQL)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	return nil
}
