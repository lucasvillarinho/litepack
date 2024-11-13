package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// Interval represents a cron schedule time.
type Interval string

const (
	EveryMinute    Interval = "*/1 * * * *"  // Run every minute
	Every5Minutes  Interval = "*/5 * * * *"  // Run every 5 minutes
	Every10Minutes Interval = "*/10 * * * *" // Run every 10 minutes
	Every15Minutes Interval = "*/15 * * * *" // Run every 15 minutes
	Every30Minutes Interval = "*/30 * * * *" // Run every 30 minutes
	EveryHour      Interval = "@hourly"      // Run every hour
)

// Scheduler is an interface for scheduling tasks.
type Scheduler interface {
	Task(scheduleTime Interval, task func() error) error
	GetTimezone() *time.Location
	Stop()
	HasTasks() bool
}

type cronScheduler interface {
	AddFunc(spec string, cmd func()) (int, error)
	Start()
	Stop()
	Entries() []cron.Entry
}

// scheduler is a simple cron scheduler.
type scheduler struct {
	timezone *time.Location
	cron     cronScheduler
}

// NewScheduler creates a new scheduler instance with the given timezone.
//
// Parameters:
//   - timezone: the timezone for the scheduler
//
// Returns:
//   - Scheduler: the scheduler instance
//   - error: an error if the operation failed
func NewScheduler(timezone *time.Location) (Scheduler, error) {
	if timezone == nil {
		return nil, fmt.Errorf("timezone cannot be nil")
	}

	cron := &cronAdapter{cron: cron.New(cron.WithLocation(timezone))}

	schedule := &scheduler{
		timezone: timezone,
		cron:     cron,
	}

	return schedule, nil
}

// Task schedules a task to run at a given schedule.
// The task is executed in the provided timezone.
//
// Parameters:
//   - scheduleTime: the schedule time for the task
//   - timezone: the timezone for the task
//   - task: the task to execute
//
// Returns:
//   - error: an error if the operation failed
func (sc *scheduler) Task(scheduleTime Interval, task func() error) error {
	_, err := sc.cron.AddFunc(string(scheduleTime), func() {
		if err := task(); err != nil {
			fmt.Printf("Error executing scheduled task: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	return nil
}

// GetTimezone returns the timezone of the scheduler.
func (sc *scheduler) GetTimezone() *time.Location {
	return sc.timezone
}

// Start starts the scheduler.
func (sc *scheduler) Start() {
	sc.cron.Start()
}

// Stop stops the scheduler.
func (sc *scheduler) Stop() {
	sc.cron.Stop()
}

// HasTasks returns true if there are tasks scheduled.
func (sc *scheduler) HasTasks() bool {
	return len(sc.cron.Entries()) > 0
}
