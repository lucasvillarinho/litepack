package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lucasvillarinho/litepack/cache/queries"
)

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
	err := ch.Database.ExecWithTx(ctx, func(tx *sql.Tx) error {
		err := ch.purgeEntriesByPercentage(ctx, tx, ch.purgePercent)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("purging cache: %w", err)
	}

	err = ch.Database.Vacuum(ctx)
	if err != nil {
		return fmt.Errorf("vacuuming cache: %w", err)
	}

	return nil
}

// PurgeExpiredItems removes expired items from the cache.
//
// Parameters:
//   - ctx: context.Context to handle cancellations or timeouts
//
// Returns:
//   - error: any error encountered during the operation
func (ch *cache) PurgeExpiredItems(ctx context.Context) error {
	now := ch.timeSource.Now().In(ch.timeSource.Timezone)
	err := ch.queries.DeleteExpiredCache(ctx, now)
	if err != nil {
		return fmt.Errorf("purging expired cache: %w", err)
	}
	return nil
}

// purgeEntriesByPercentage deletes a percentage of the cache entries.
func (ch *cache) purgeEntriesByPercentage(ctx context.Context, tx *sql.Tx, percent float64) error {
	if percent < 0 || percent > 1 {
		return fmt.Errorf("invalid percentage: %f", percent)
	}

	queriesWityTx := queries.New(tx)

	totalEntries, err := queriesWityTx.CountCacheEntries(ctx)
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

// purgeExpiredItensCache clears expired cache items periodically.
func (ch *cache) purgeExpiredItensCache(ctx context.Context) {
	task := func() {
		err := ch.queries.DeleteExpiredCache(ctx, time.Now().In(ch.timeSource.Timezone))
		if err != nil {
			err = fmt.Errorf("deleting expired cache: %w", err)
			ch.logger.Error(ctx, err.Error())
			return
		}
	}

	_, err := ch.cron.AddAndExec(string(ch.syncInterval), task)
	if err != nil {
		err = fmt.Errorf("adding cron task: %w", err)
		ch.logger.Error(ctx, err.Error())
		return
	}

	ch.cron.Start()
}
