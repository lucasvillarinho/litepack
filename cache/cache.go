package cache

import (
	"database/sql"
	"fmt"
	"log/slog"
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

// CacheOption is a function that configures a cache instance.
type Option func(*cache)

// WithClearInterval sets a custom sync interval for the cache.
func WithClearInterval(interval schedule.Interval) Option {
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

	err := c.setDriver()
	if err != nil {
		return nil, fmt.Errorf("error setting driver: %w", err)
	}

	err = c.setupTable()
	if err != nil {
		return nil, fmt.Errorf("error setting up cache table: %w", err)
	}

	err = c.setupSyncClearByTTL()
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

// startSyncClearByTTL sets up a schedule to clear expired cache items.
//
// Parameters:
//   - scheduler: the scheduler to use
//   - clearExpiredItems: the function to clear expired cache items
func startSyncClearByTTL(scheduler schedule.Scheduler, clearExpiredItems func() error) {
	go func() {
		err := scheduler.Task(schedule.EveryMinute, clearExpiredItems)
		if err != nil {
			slog.Error("Failed to schedule cache clear task", slog.String("error", err.Error()))
		}
	}()
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
func (ch *cache) createCacheTable() error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS cache (
        key TEXT PRIMARY KEY,
        value BLOB,
        expires_at TIMESTAMP,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
// Returns:
//   - error: an error if the operation failed
func (ch *cache) setDriver() error {
	driverFactory := drivers.NewDriverFactory()

	engine, err := driverFactory.GetDriver(ch.drive, ch.dsn)
	if err != nil {
		return fmt.Errorf("error getting driver: %w", err)
	}
	ch.engine = engine

	return nil
}

// SetupTable creates the cache table with custom configuration.

// Returns:
//   - error: an error if the operation failed
func (ch *cache) setupTable() error {
	err := ch.createCacheTable()
	if err != nil {
		return err
	}

	err = ch.createIndex()
	if err != nil {
		return err
	}

	err = ch.setWalMode()
	if err != nil {
		return err
	}

	err = ch.setCacheSize()
	if err != nil {
		return err
	}

	err = ch.setSynchronousMode()
	if err != nil {
		return err
	}

	return nil
}

// createIndex creates an index on the cache table for the key column.
//
// Parameters:
//   - db: the database handle
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) createIndex() error {
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
// Returns:
//
//   - error: an error if the operation failed
func (ch *cache) setCacheSize() error {
	pages := ch.cacheSize / 4096

	query := fmt.Sprintf("PRAGMA cache_size = %d;", pages)

	_, err := ch.engine.Execute(query)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	return nil
}

// setWalMode enables Write-Ahead Logging (WAL) mode for the database.
// WAL mode allows for concurrent reads and writes to the database.
// WAL mode is recommended for high-traffic applications.
//
// Parameters:
//   - db: the database handle
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) setWalMode() error {
	_, err := ch.engine.Execute("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("enabling WAL mode: %w", err)
	}
	return nil
}

// setSynchronousMode sets the synchronous mode for the database.
// Synchronous mode determines how often the database writes to disk.
func (ch *cache) setSynchronousMode() error {
	_, err := ch.engine.Execute("PRAGMA synchronous = NORMAL;")
	if err != nil {
		return fmt.Errorf("setting synchronous mode: %w", err)
	}
	return nil
}

// setupSyncClearByTTL sets up a schedule to clear expired cache items.
//
// Returns:
//
//   - error: an error if the operation failed
func (ch *cache) setupSyncClearByTTL() error {
	scheduler, err := schedule.NewScheduler(ch.timezone)
	ch.scheduler = scheduler
	startSyncClearByTTL(scheduler, ch.clearExpiredItems)

	return err
}
