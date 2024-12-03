package cache

import (
	"time"

	"github.com/lucasvillarinho/litepack/internal/cron"
)

// CacheOption is a function that configures a cache instance.
type Option func(*cache)

// WithSyncInterval sets a custom sync interval for the cache.
// The sync interval determines how often the cache is synchronized with the database.
func WithSyncInterval(interval cron.Interval) Option {
	return func(c *cache) {
		c.syncInterval = interval
	}
}

// WithPath sets the path to the cache database.
// The cache is automatically created if it does not exist.
func WithPath(path string) Option {
	return func(c *cache) {
		c.path = path
	}
}

// WithTimezone sets a custom timezone for the cache.
func WithTimezone(timezone *time.Location) Option {
	return func(c *cache) {
		c.timeSource.Timezone = timezone
	}
}

// WithPurgePercent sets the percentage of cache entries to delete when purging.
func WithPurgePercent(percent float64) Option {
	return func(c *cache) {
		c.purgePercent = percent
	}
}

// WithPurgeTimeout sets the timeout for purging cache entries.
func WithPurgeTimeout(timeout time.Duration) Option {
	return func(c *cache) {
		c.purgeTimeout = timeout
	}
}
