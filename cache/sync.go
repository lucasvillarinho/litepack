package cache

import (
	"fmt"

	"github.com/lucasvillarinho/litepack/schedule"
)

// startSyncClearByTTL starts the cache cleaner task.
//
// Parameters:
//   - ch: the cache handle
//
// Returns:
//   - error: an error if the operation failed
func startSyncClearByTTL(ch *cache) error {
	scheduler, err := schedule.NewScheduler(ch.timezone)
	ch.scheduler = scheduler
	createTaskCleaner(scheduler, ch.clearExpiredItems)
	return err
}

// createTaskCleaner creates a task to clean expired cache items.
func createTaskCleaner(scheduler schedule.Scheduler, clearExpiredItems func() error) {
	go func() {
		err := scheduler.Task(schedule.EveryMinute, clearExpiredItems)
		if err != nil {
			fmt.Printf("Error scheduling task: %v\n", err)
		}
	}()
}
