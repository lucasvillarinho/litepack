package cache

import "github.com/lucasvillarinho/litepack/internal/cron"

// CacheOption is a function that configures a cache instance.
type Option func(*cache)

// WithClearInterval sets a custom sync interval for the cache.
func WithClearInterval(interval cron.Interval) Option {
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
