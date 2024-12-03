package cache

import (
	"context"
	"fmt"

	"github.com/lucasvillarinho/litepack/cache/queries"
)

// setupCache sets up the cache with the given configuration.
func (ch *cache) setupCacheTable(ctx context.Context) error {
	// Set up the cache queries.
	ch.queries = queries.New(ch.Database.GetEngine(ctx))

	// create the cache table if it does not exist
	err := ch.queries.CreateCacheDatabase(ctx)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	// create the index key_expires_at if it does not exist
	sqlIndexKeyExpiresAt := `CREATE INDEX IF NOT EXISTS idx_key_expires_at ON cache(key, expires_at)`
	err = ch.Database.Exec(ctx, sqlIndexKeyExpiresAt)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}

	return nil
}

// setupCacheDatabase sets up the cache database with the given configuration.
func (ch *cache) setupCacheDatabase(ctx context.Context) error {
	err := ch.Database.SetJournalModeWal(ctx)
	if err != nil {
		return fmt.Errorf("setting journal mode: %w", err)
	}

	err = ch.Database.SetPageSize(ctx, ch.pageSize)
	if err != nil {
		return fmt.Errorf("setting page size: %w", err)
	}

	err = ch.Database.SetCacheSize(ctx, ch.cacheSize)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	// Max page count is the maximum number of pages in the database file.
	err = ch.Database.SetMaxPageCount(ctx, ch.maxDBSize/ch.pageSize)
	if err != nil {
		return fmt.Errorf("setting max page count: %w", err)
	}

	return nil
}
