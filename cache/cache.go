package cache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
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
	query, args, err := squirrel.
		Insert("cache").
		Columns("key", "value", "expires_at", "last_accessed_at").
		Values(key, value, time.Now().Add(ttl).In(ch.timezone), time.Now().In(ch.timezone)).
		Suffix("ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	_, err = ch.engine.Execute(query, args...)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}

	return nil
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

	query, args, err := squirrel.
		Select("value").
		From("cache").
		Where(squirrel.And{
			squirrel.Eq{"key": key},
			squirrel.Gt{"expires_at": time.Now().In(ch.timezone)},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	err = ch.engine.QueryRow(query, args...).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting value: %w", err)
	}

	err = ch.updateLastAccessedAt(key)
	if err != nil {
		fmt.Printf("error updating last accessed at: %v\n", err)
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
	query, args, err := squirrel.
		Delete("cache").
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	_, err = ch.engine.Execute(query, args...)
	if err != nil {
		fmt.Println("error deleting key", err)
		return fmt.Errorf("deleting key: %w", err)
	}
	return err
}

// updateLastAccessedAt updates the last accessed at timestamp for a cache entry.
func (ch *cache) updateLastAccessedAt(key string) error {
	updateQuery, updateArgs, err := squirrel.
		Update("cache").
		Set("last_accessed_at", time.Now().In(ch.timezone)).
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return fmt.Errorf("building update query: %w", err)
	}

	_, err = ch.engine.Execute(updateQuery, updateArgs...)
	if err != nil {
		return fmt.Errorf("updating last_accessed_at: %w", err)
	}

	return nil
}

// Purge deletes a percentage of the cache entries.
// The entries are deleted in ascending order of last accessed at timestamp (LRU).
// The percentage must be between 0 and 1.
//
// Parameters:
//   - percent: the percentage of entries to delete
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) Purge(percent float64) error {
	if percent < 0 || percent > 1 {
		return fmt.Errorf("invalid percentage: %f", percent)
	}

	var totalEntries int
	countQuery, args, err := squirrel.
		Select("COUNT(*)").
		From("cache").
		ToSql()
	if err != nil {
		return fmt.Errorf("building count query: %w", err)
	}

	err = ch.engine.QueryRow(countQuery, args...).Scan(&totalEntries)
	if err != nil {
		return fmt.Errorf("failed to count entries: %w", err)
	}

	totalEntriesToDelete := int(float64(totalEntries) * percent)
	if totalEntriesToDelete == 0 {
		return nil
	}

	subQuery, subArgs, err := squirrel.
		Select("key").
		From("cache").
		OrderBy("last_accessed_at ASC").
		Limit(uint64(totalEntriesToDelete)).
		ToSql()
	if err != nil {
		return fmt.Errorf("building subquery: %w", err)
	}

	deleteQuery, deleteArgs, err := squirrel.
		Delete("cache").
		Where(fmt.Sprintf("key IN (%s)", subQuery), subArgs...).
		ToSql()
	if err != nil {
		return fmt.Errorf("building delete query: %w", err)
	}

	_, err = ch.engine.Execute(deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("executing purge query: %w", err)
	}

	return nil
}

// Vacuum reclaims unused database space.
//
// Returns:
//
//   - error: an error if the operation failed
//
// WARNING: This operation can be slow for large databases.
func (ch *cache) Vacuum() error {
	_, err := ch.engine.Execute("VACUUM;")
	if err != nil {
		return fmt.Errorf("error vacuuming: %w", err)
	}
	return nil
}

// clearExpiredItems Deletes all cache entries that have expired.
//
// Returns:
//   - error: an error if the operation failed
func (ch *cache) clearExpiredItems() error {
	query, args, err := squirrel.
		Delete("cache").
		Where(squirrel.LtOrEq{"expires_at": time.Now().In(ch.timezone)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	_, err = ch.engine.Execute(query, args...)
	if err != nil {
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
