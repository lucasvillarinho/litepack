package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockCron struct {
	addFuncErr error
}

func (m *MockCron) AddFunc(schedule string, cmd func()) (int, error) {
	if m.addFuncErr != nil {
		return 0, m.addFuncErr
	}
	go cmd()
	return 1, nil
}

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
