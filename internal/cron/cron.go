package cron

import (
	"time"

	crf "github.com/robfig/cron/v3"
)

// Interval represents a cron schedule interval.
type Interval string

const (
	EveryMinute    Interval = "*/1 * * * *"  // Run every minute
	Every5Minutes  Interval = "*/5 * * * *"  // Run every 5 minutes
	Every10Minutes Interval = "*/10 * * * *" // Run every 10 minutes
	Every15Minutes Interval = "*/15 * * * *" // Run every 15 minutes
	Every30Minutes Interval = "*/30 * * * *" // Run every 30 minutes
	EveryHour      Interval = "@hourly"      // Run every hour
)

type Cron interface {
	Add(schedule string, task func()) (crf.EntryID, error)
	Remove(entryID crf.EntryID)
	Start()
	Stop()
}

type cron struct {
	cron *crf.Cron
}

// New creates a new Cron instance with a specified timezone.
//
// Parameters:
//   - timezone: the timezone for scheduling tasks (default is UTC if nil)
//
// Returns:
//   - *Cron: the Cron facade instance
func New(timezone *time.Location) Cron {
	if timezone == nil {
		timezone = time.UTC
	}

	return &cron{
		cron: crf.New(crf.WithLocation(timezone)),
	}

}

// Add schedules a task to run at the specified interval.
//
// Parameters:
//   - schedule: the cron schedule string (e.g., "*/5 * * * *")
//   - task: the function to execute
//
// Returns:
//   - cron.EntryID: the ID of the scheduled task
//   - error: if the schedule string or task is invalid
func (c *cron) Add(schedule string, task func()) (crf.EntryID, error) {
	return c.cron.AddFunc(schedule, task)
}

// Remove cancels a scheduled task by its EntryID.
//
// Parameters:
//   - entryID: the ID of the task to remove
func (c *cron) Remove(entryID crf.EntryID) {
	c.cron.Remove(entryID)
}

// Start begins the execution of scheduled tasks.
func (c *cron) Start() {
	c.cron.Start()
}

// Stop halts the execution of scheduled tasks.
func (c *cron) Stop() {
	c.cron.Stop()
}
