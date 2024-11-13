package schedule

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewScheduler(t *testing.T) {
	t.Run("should create a scheduler with a valid timezone", func(t *testing.T) {
		timezone := time.UTC

		scheduler, err := NewScheduler(timezone)

		assert.NoError(t, err)
		assert.NotNil(t, scheduler)
		assert.Equal(t, timezone, scheduler.GetTimezone())
	})

	t.Run("should return an error when timezone is nil", func(t *testing.T) {
		var timezone *time.Location = nil

		scheduler, err := NewScheduler(timezone)

		assert.Error(t, err)
		assert.Nil(t, scheduler)
		assert.EqualError(t, err, "timezone cannot be nil")
	})
}

func TestTask(t *testing.T) {
	t.Run("should schedule a task successfully", func(t *testing.T) {
		mock := &mockCron{}
		timezone := time.UTC
		scheduler := &scheduler{
			timezone: timezone,
			cron:     mock,
		}
		defer scheduler.Stop()

		task := func() error {
			fmt.Println("Task executed")
			return nil
		}

		err := scheduler.Task(EveryMinute, task)

		assert.NoError(t, err, "Expected no error when scheduling a task")
		assert.Len(t, mock.addFuncCalls, 1, "Expected one task to be scheduled")
		assert.Equal(
			t,
			"*/1 * * * *",
			mock.addFuncCalls[0].spec,
			"Expected task to be scheduled with the correct spec",
		)
		assert.NotNil(t, mock.addFuncCalls[0].cmd, "Expected scheduled task command to be not nil")
	})

	t.Run("should return an error when AddFunc fails", func(t *testing.T) {
		mock := &mockCron{
			addFuncErr: fmt.Errorf("mock AddFunc error"),
		}
		timezone := time.UTC
		scheduler := &scheduler{
			timezone: timezone,
			cron:     mock,
		}
		defer scheduler.Stop()

		task := func() error {
			fmt.Println("Task executed")
			return nil
		}

		err := scheduler.Task(EveryMinute, task)

		assert.Error(t, err, "Expected an error when AddFunc fails")
		assert.EqualError(t, err, "failed to schedule task: mock AddFunc error")
		assert.Len(t, mock.addFuncCalls, 0, "Expected no tasks to be scheduled when AddFunc fails")
	})
}
