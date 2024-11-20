package database

import "github.com/lucasvillarinho/litepack/database/drivers"

// WithEngine sets the database engine.
func WithEngine(engine drivers.Driver) Option {
	return func(db *database, cfg *config) {
		db.engine = engine
	}
}

// WithCacheSize sets the cache size.
func WithCacheSize(size int) Option {
	return func(db *database, cfg *config) {
		cfg.cacheSize = size
	}
}

// WithPageSize sets the page size.
func WithPageSize(size int) Option {
	return func(db *database, cfg *config) {
		cfg.pageSize = size
	}
}

// WithDBSize sets the database size.
func WithDBSize(size int) Option {
	return func(db *database, cfg *config) {
		cfg.dbSize = size
	}
}
