package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lucasvillarinho/litepack/cache/queries"
	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/internal/cron"
	"github.com/lucasvillarinho/litepack/internal/helpers"
	"github.com/lucasvillarinho/litepack/internal/log"
)

// timeSource is used to get the current time.
type timeSource struct {
	Timezone *time.Location
	Now      func() time.Time // Now returns the current time.
}

// ErrKeyNotFound is returned when a key is not found in the cache.
var ErrKeyNotFound = fmt.Errorf("key not found")

// cache is a simple key-value store backed by an SQLite database.
type cache struct {
	timeSource timeSource
	cron       cron.Cron
	database.Database
	logger log.Logger

	// purge configuration, puging is used to delete cache entries
	purgePercent float64
	purgeTimeout time.Duration
	syncInterval cron.Interval

	// database configuration
	path      string
	dbName    string
	cacheSize int
	pageSize  int
	maxDBSize int
	queries   *queries.Queries
}

// Cache is a simple key-value store backed by an SQLite database.
type Cache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	database.Database
}

// NewCache creates a new cache instance with the given name and applies any provided options.
// The cache is backed by an SQLite database.
//
// The cache is automatically created if it does not exist.
//
// Parameters:
//   - ctx: the context
//   - opts: the cache options
//
// Returns:
//   - cache: the cache instance
//   - error: an error if the operation failed
//
// Configuration defaults:
//   - syncInterval: 1 second
//   - timezone: UTC
//
// Configuration options:
//   - WithSyncInterval: sets a custom sync interval for the cache.
//   - WithPath: sets the path to the cache database.
//   - WithTimezone: sets a custom timezone for the cache.
//   - WithPurgePercent: sets the percentage of cache entries to purge.
//   - WithPurgeTimeout: sets the timeout for purging cache entries.
//   - WithDBOptions: sets the database options.
//
// Example:
//
//	cache, err := cache.NewCache(ctx)
//	if err != nil {
//		panic(err)
//	}
func NewCache(ctx context.Context, opts ...Option) (Cache, error) {
	c := &cache{
		purgePercent: 0.2,              // 20%
		purgeTimeout: 30 * time.Second, // 30 seconds
		dbName:       "lpack_cache.db",
		cacheSize:    64 * 1024 * 1024,  // 64 MB
		pageSize:     4096,              // 4 KB
		maxDBSize:    512 * 1024 * 1024, // 512 MB
		timeSource: timeSource{
			Timezone: time.UTC,
			Now:      time.Now,
		},
		syncInterval: cron.EveryMinute,
		cron:         cron.New(time.UTC),
	}

	for _, opt := range opts {
		opt(c)
	}

	/// database is used to store cache entries
	cacheDB, err := database.NewDatabase(ctx, c.path, c.dbName)
	if err != nil {
		return nil, err
	}
	c.Database = cacheDB

	// logger is used to log errors when setting cache entries
	logger, err := log.NewLogger(ctx, c.Database)
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}
	c.logger = logger

	// create database if it does not exist and apply database options
	err = c.setupCacheDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up cache: %w", err)
	}

	// create cache table if it does not exist and apply indexes
	err = c.setupCacheTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up cache queries: %w", err)
	}

	// start the cron job to clear expired cache items
	go c.purgeExpiredItensCache(ctx)

	return c, nil
}

// Set sets a key-value pair in the cache with the given TTL.
// If the key already exists, it is updated with the new value and TTL.
// The key-value pair is automatically removed from the cache after the TTL expires.
//
// Parameters:
//   - ctx: the context
//   - key: the cache key
//   - value: the cache value
//   - ttl: the time-to-live for the cache entry (in seconds)
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	cache, err := cache.NewCache(ctx)
//	defer cache.Close(ctx)
//
//	err := cache.Set(ctx, "key", "test", 10*time.Second) // no error
//	if err != nil {
//		return err
//	}
func (ch *cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	attempt := 0
	maxAttempts := 2

	retryFunc := func() error {
		attempt++
		now := ch.timeSource.Now().In(ch.timeSource.Timezone)
		expiresAt := now.Add(ttl)

		params := queries.UpsertCacheParams{
			Key:            key,
			Value:          []byte(value),
			ExpiresAt:      expiresAt,
			LastAccessedAt: now,
		}

		if err := ch.queries.UpsertCache(context.Background(), params); err != nil {
			// If the database is full, purge the cache and try again.

			if database.IsDBFullError(err) && attempt < maxAttempts {
				if err = ch.PurgeItens(ctx); err != nil {
					return fmt.Errorf("error purging cache: %w", err)
				}
			}
			return fmt.Errorf("error setting cache: %w", err)
		}

		return nil
	}

	// Retry the set operation if the database is full
	if err := helpers.Retry(ctx, retryFunc, maxAttempts); err != nil {
		return err
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
//   - string: the cache value
//   - error: an error if the operation failed
//
// Example:
//
//	cache, err := cache.NewCache(ctx)
//	defer cache.Close(ctx)
//
//	value, err := cache.Get(ctx, "key") // value: test
//	if err != nil {
//		return err
//	}
func (ch *cache) Get(ctx context.Context, key string) (string, error) {
	paramsGet := queries.GetValueParams{
		Key:       key,
		ExpiresAt: time.Now().In(ch.timeSource.Timezone),
	}

	value, err := ch.queries.GetValue(ctx, paramsGet)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrKeyNotFound
		}

		return "", fmt.Errorf("error getting value: %w", err)
	}

	paramsUpdate := queries.UpdateLastAccessedAtParams{
		LastAccessedAt: time.Now().In(ch.timeSource.Timezone),
		Key:            key,
	}

	err = ch.queries.UpdateLastAccessedAt(ctx, paramsUpdate)
	if err != nil {
		fmt.Printf("error updating last accessed at: %v\n", err)
	}

	return string(value), nil
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
//
// Example:
//
//	cache, err := cache.NewCache(ctx)
//	defer cache.Close(ctx)
//
//	err := cache.Del(ctx, "key") // no error
func (ch *cache) Del(ctx context.Context, key string) error {
	err := ch.queries.DeleteKey(ctx, key)
	if err != nil {
		return fmt.Errorf("deleting key: %w", err)
	}

	return nil
}

// Close closes the cache and stops jobs.
//
// Parameters:
//   - ctx: the context
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	cache, err := cache.NewCache(ctx)
//	defer cache.Close(ctx)
func (ch *cache) Close(ctx context.Context) error {
	ch.cron.Stop()
	return ch.Database.Close(ctx)
}
