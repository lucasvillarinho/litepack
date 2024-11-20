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
	"github.com/lucasvillarinho/litepack/internal/helpers"
	"github.com/lucasvillarinho/litepack/internal/schedule"
)

// cache is a simple key-value store backed by an SQLite database.
type cache struct {
	scheduler    schedule.Scheduler
	timezone     *time.Location
	queries      *queries.Queries
	syncInterval schedule.Interval
	dsn          string
	purgePercent float64
	purgeTimeout time.Duration
	sync.RWMutex
	database.Database
}

type Cache interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
	database.Database
}

// NewCache creates a new cache instance with the given name and applies any provided options.
// The cache is backed by an SQLite database.
//
// The path is used to create a database file with the format "<path>_lpack_cache.db".
// The cache is automatically created if it does not exist.
//
// Parameters:
//   - ctx: the context
//   - path: the path to the cache database
//   - opts: the cache options
//
// Returns:
//   - cache: the cache instance
//   - error: an error if the operation failed
//
// Configuration defaults:
//   - syncInterval: 1 second
//   - timezone: UTC

// Configuration options:
//   - WithSyncInterval: sets a custom sync interval for the cache.
//   - WithTimezone: sets a custom timezone for the cache.
func NewCache(ctx context.Context, path string, opts ...Option) (Cache, error) {
	c := &cache{
		syncInterval: schedule.EveryMinute,
		timezone:     time.UTC,
		purgePercent: 0.2, // 20%
		purgeTimeout: 30 * time.Second,
	}

	dsn, err := helpers.CreateDSN(path)
	if err != nil {
		return nil, fmt.Errorf("error creating DSN: %w", err)
	}
	c.dsn = dsn

	for _, opt := range opts {
		opt(c)
	}

	// Create the cache instance.
	db, err := database.NewDatabase(ctx, c.dsn,
		database.WithDBSize(128*1024*1024),
		database.WithCacheSize(128*1024*1024),
		database.WithPageSize(4096))
	if err != nil {
		return nil, fmt.Errorf("error creating database: %w", err)
	}
	c.Database = db

	// Set up the cache queries.
	c.queries = queries.New(c.Database.GetEngine(ctx))

	// Set up the cache table.
	err = c.createCacheTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up table: %w", err)
	}

	return c, nil
}

// setupTable creates the cache table if it does not exist.
func (ch *cache) createCacheTable(ctx context.Context) error {
	err := ch.queries.CreateCacheDatabase(ctx)
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
			if ch.Database.IsDBFullError(err) {
				if err = ch.PurgeItens(ctx); err != nil {
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

// PurgeItens deletes a percentage of the cache entries.
// The entries are deleted in ascending order of last accessed at timestamp (LRU).
// The percentage must be between 0 and 1.
//
// Parameters:
//   - ctx: the context
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) PurgeItens(ctx context.Context) error {
	return ch.Database.ExecWithTx(ctx, func(tx *sql.Tx) error {
		err := ch.purgeEntriesByPercentage(ctx, tx, ch.purgePercent)
		if err != nil {
			return fmt.Errorf("error purging cache: %w", err)
		}

		err = ch.Database.Vacuum(ctx, tx)
		if err != nil {
			return fmt.Errorf("error vacuuming cache: %w", err)
		}

		return nil
	})
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

// purgeEntriesByPercentage deletes a percentage of the cache entries.
func (ch *cache) purgeEntriesByPercentage(ctx context.Context, tx *sql.Tx, percent float64) error {
	if percent < 0 || percent > 1 {
		return fmt.Errorf("invalid percentage: %f", percent)
	}

	queriesWityTx := queries.New(tx)

	totalEntries, err := queriesWityTx.CountEntries(ctx)
	if err != nil {
		return fmt.Errorf("count entries: %w", err)
	}

	// Calculate the number of entries to delete.
	totalEntriesToDelete := int64(float64(totalEntries) * percent)
	if totalEntriesToDelete == 0 {
		return nil
	}

	err = queriesWityTx.DeleteKeysByLimit(ctx, totalEntriesToDelete)
	if err != nil {
		return fmt.Errorf("delete entries: %w", err)
	}

	return nil
}
