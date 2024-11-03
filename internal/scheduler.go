package internal

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduleTime represents a cron schedule time.
type ScheduleTime string

const (
	Every1Minutes  ScheduleTime = "*/1 * * * *"  // Run every minute
	Every5Minutes  ScheduleTime = "*/5 * * * *"  // Run every 5 minutes
	Every10Minutes ScheduleTime = "*/10 * * * *" // Run every 10 minutes
	Every15Minutes ScheduleTime = "*/15 * * * *" // Run every 15 minutes
	Every30Minutes ScheduleTime = "*/30 * * * *" // Run every 30 minutes
	EveryHour      ScheduleTime = "@hourly"      // Run every hour
)

// StartAutoCleanup schedules a task to run at a given schedule.
// The task is executed in the provided timezone.
//
// Parameters:
//   - scheduleTime: the schedule time for the task
//   - timezone: the timezone for the task
//   - task: the task to execute
//
// Returns:
//   - error: an error if the operation failed
func ScheduleFunction(scheduleTime ScheduleTime, timezone *time.Location, task func() error) error {
	cleaner := cron.New(cron.WithLocation(timezone))
	_, err := cleaner.AddFunc(string(scheduleTime), func() {
		if err := task(); err != nil {
			fmt.Printf("Error executing scheduled task: %v\n", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	cleaner.Start()
	return nil
}
