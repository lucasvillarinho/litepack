package cache

import (
	"context"
	"fmt"

	"github.com/lucasvillarinho/litepack/internal/schedule"
)

// startSyncClearByTTL starts the cache cleaner task.
//
// Parameters:
//   - ch: the cache handle
//
// Returns:
//   - error: an error if the operation failed
func startSyncClearByTTL(ctx context.Context, ch *cache) error {
	scheduler, err := schedule.NewScheduler(ch.timezone)
	ch.scheduler = scheduler
	createTaskCleaner(ctx, scheduler, ch.clearExpiredItems)
	return err
}

// createTaskCleaner creates a task to clean expired cache items.
func createTaskCleaner(
	ctx context.Context,
	scheduler schedule.Scheduler,
	clearExpiredItems func(context.Context) error,
) {
	go func() {
		err := scheduler.Task(ctx, schedule.EveryMinute, clearExpiredItems)
		if err != nil {
			fmt.Printf("Error scheduling task: %v\n", err)
		}
	}()
}
