package cache

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lucasvillarinho/litepack/cache/queries"
	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/database/drivers"
	"github.com/lucasvillarinho/litepack/internal/helpers"
	"github.com/lucasvillarinho/litepack/schedule"
)

// cache is a simple key-value store backed by an SQLite database.
type cache struct {
	scheduler    schedule.Scheduler
	engine       drivers.Driver
	timezone     *time.Location
	queries      *queries.Queries
	syncInterval schedule.Interval
	dsn          string
	drive        drivers.DriverType
	dbSize       int
	cacheSize    int
	pageSize     int
	purgePercent float64
	purgeTimeout time.Duration
	sync.RWMutex
}

type Cache interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
	Close() error
	Destroy() error
}

// NewCache creates a new cache instance with the given name and applies any provided options.
// The cache is backed by an SQLite database.
// The path is used to create a database file with the format "<name>_lpack_cache.db".
// The cache is automatically created if it does not exist.
//
// Parameters:
//   - ctx: the context
//   - path: the path to the cache database
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
func NewCache(ctx context.Context, path string, opts ...Option) (Cache, error) {
	c := &cache{
		dsn:          fmt.Sprintf("%s_lpack_cache.db", path),
		syncInterval: schedule.EveryMinute,
		timezone:     time.UTC,
		drive:        drivers.DriverMattn,
		cacheSize:    128 * 1024 * 1024, // 128 MB
		dbSize:       128 * 1024 * 1024, // 128 MB
		pageSize:     4096,
		purgePercent: 0.2, // 20%
		purgeTimeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	err := c.setupEngine(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up engine: %w", err)
	}

	err = c.setupDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up database: %w", err)
	}

	err = c.setupTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up table: %w", err)
	}

	err = startSyncClearByTTL(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("error setting up sync clear by TTL: %w", err)
	}

	return c, nil
}

// setupDatabase sets up the cache database with the given configuration.
func (ch *cache) setupDatabase(ctx context.Context) error {
	// Set journal mode to WAL
	if _, err := ch.engine.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Set synchronous mode to NORMAL
	if _, err := ch.engine.ExecContext(ctx, "PRAGMA synchronous = NORMAL;"); err != nil {
		return fmt.Errorf("setting synchronous mode: %w", err)
	}

	// Set the maximum page count for the database
	if _, err := ch.engine.ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d;", ch.dbSize/ch.pageSize)); err != nil {
		return fmt.Errorf("setting max page count: %w", err)
	}

	// Set the page size in bytes
	if _, err := ch.engine.ExecContext(ctx, fmt.Sprintf("PRAGMA page_size = %d;", ch.pageSize)); err != nil {
		return fmt.Errorf("setting page size: %w", err)
	}

	// Set the cache size in pages
	if _, err := ch.engine.ExecContext(ctx, fmt.Sprintf("PRAGMA cache_size = %d;", ch.cacheSize/ch.pageSize)); err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	return nil
}

// SetupEngine creates a new database engine with the given driver and DSN.
func (ch *cache) setupEngine(_ context.Context) error {
	engine, err := drivers.NewDriverFactory().GetDriver(ch.drive, ch.dsn)
	if err != nil {
		return fmt.Errorf("error creating driver: %w", err)
	}
	ch.engine = engine

	ch.queries = queries.New(ch.engine)

	return nil
}

// setupTable creates the cache table if it does not exist.
func (ch *cache) setupTable(ctx context.Context) error {
	err := ch.queries.CreateDatabase(ctx)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	return nil
}

// Set sets a key-value pair in the cache with the given TTL.
// If the key already exists, it is updated with the new value and TTL.
// The key-value pair is automatically removed from the cache after the TTL expires.
//
// Parameters:
//   - ctx: the context
//   - key: the cache key
//   - value: the cache value
//   - ttl: the time-to-live for the cache entry
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	retryFunc := func() error {
		now := time.Now().In(ch.timezone)
		expiresAt := now.Add(ttl)

		params := queries.UpsertCacheParams{
			Key:            key,
			Value:          value,
			ExpiresAt:      expiresAt,
			LastAccessedAt: now,
		}

		if err := ch.queries.UpsertCache(context.Background(), params); err != nil {
			// If the database is full, purge the cache and try again.
			if database.IsDatabaseFullError(err) {
				if err = ch.PurgeDB(ctx); err != nil {
					return fmt.Errorf("error purging cache: %w", err)
				}
			}
			return fmt.Errorf("error executing query: %w", err)
		}

		return nil
	}

	if err := helpers.Retry(ctx, retryFunc, 2); err != nil {
		return fmt.Errorf("error retrying set: %w", err)
	}
	return nil
}

// PurgeDB deletes a percentage of the cache entries.
// The entries are deleted in ascending order of last accessed at timestamp (LRU).
// The percentage must be between 0 and 1.
//
// Parameters:
//   - ctx: the context
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) PurgeDB(ctx context.Context) error {
	tx, err := ch.engine.Begin()
	if err != nil {
		return fmt.Errorf("error to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			err = fmt.Errorf("transaction failed due to panic: %v", p)
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err := ch.PurgeWithTransaction(ctx, ch.purgePercent, tx); err != nil {
		return fmt.Errorf("error purging cache: %w", err)
	}

	if err := ch.VacuumWithTransaction(tx); err != nil {
		return fmt.Errorf("error vacuuming cache: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error to commit transaction: %w", err)
	}
	return nil
}

// Get retrieves a value from the cache by key.
//
// Parameters:
//   - ctx: the context
//   - key: the cache key
//
// Returns:
//   - []byte: the cache value
//   - error: an error if the operation failed
func (ch *cache) Get(ctx context.Context, key string) ([]byte, error) {
	paramsGet := queries.GetValueParams{
		Key:       key,
		ExpiresAt: time.Now().In(ch.timezone),
	}

	value, err := ch.queries.GetValue(ctx, paramsGet)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting value: %w", err)
	}

	paramsUpdate := queries.UpdateLastAccessedAtParams{
		LastAccessedAt: time.Now().In(ch.timezone),
		Key:            key,
	}

	err = ch.queries.UpdateLastAccessedAt(ctx, paramsUpdate)
	if err != nil {
		fmt.Printf("error updating last accessed at: %v\n", err)
	}

	return value, nil
}

// Del deletes a key-value pair from the cache.
// If the key does not exist, the operation is a no-op.
//
// Parameters:
//   - ctx: the context
//   - key: the cache key
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) Del(ctx context.Context, key string) error {
	err := ch.queries.DeleteKey(ctx, key)
	if err != nil {
		return fmt.Errorf("deleting key: %w", err)
	}

	return nil
}

// Purge deletes a percentage of the cache entries.
// The entries are deleted in ascending order of last accessed at timestamp (LRU).
// The percentage must be between 0 and 1.
//
// Parameters:
//   - ctx: the context
//   - percent: the percentage of entries to delete
//   - tx: the database transaction
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) PurgeWithTransaction(ctx context.Context, percent float64, tx *sql.Tx) error {
	if percent < 0 || percent > 1 {
		return fmt.Errorf("invalid percentage: %f", percent)
	}

	queriesWityTx := queries.New(tx)

	totalEntries, err := queriesWityTx.CountEntries(ctx)
	if err != nil {
		return fmt.Errorf("error to count entries: %w", err)
	}

	totalEntriesToDelete := int64(float64(totalEntries) * percent)
	if totalEntriesToDelete == 0 {
		return nil
	}

	err = queriesWityTx.DeleteKeys(ctx, totalEntriesToDelete)
	if err != nil {
		return fmt.Errorf("error to delete entries: %w", err)
	}

	return nil
}

// Vacuum reclaims unused database space.
//
// Returns:
//   - error: an error if the operation failed
//
// WARNING: This operation can be slow for large databases.
func (ch *cache) VacuumWithTransaction(tx *sql.Tx) error {
	_, err := tx.Exec("VACUUM;")
	if err != nil {
		return fmt.Errorf("error vacuuming: %w", err)
	}
	return nil
}

// clearExpiredItems Deletes all cache entries that have expired.
//
// Parameters:
//   - ctx: the context
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) clearExpiredItems(ctx context.Context) error {
	if err := ch.queries.DeleteExpiredCache(ctx, time.Now().In(ch.timezone)); err != nil {
		return fmt.Errorf("error clear: %w", err)
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
