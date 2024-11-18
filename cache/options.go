package cache

import (
	"time"

	"github.com/lucasvillarinho/litepack/internal/schedule"
)

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

func WithCacheSize(size int) Option {
	return func(c *cache) {
		c.cacheSize = size
	}
}
