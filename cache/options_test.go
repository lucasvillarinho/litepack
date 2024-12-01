package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/internal/cron"
)

func TestCacheOptions(t *testing.T) {
	t.Run("WithSyncInterval", func(t *testing.T) {
		c := &cache{}
		interval := cron.Every5Minutes

		WithSyncInterval(interval)(c)

		assert.Equal(t, interval, c.syncInterval, "syncInterval should be set correctly")
	})

	t.Run("WithPath", func(t *testing.T) {
		c := &cache{}
		path := "/tmp/cache.db"

		WithPath(path)(c)

		assert.Equal(t, path, c.path, "path should be set correctly")
	})

	t.Run("WithTimezone", func(t *testing.T) {
		c := &cache{}
		timezone := time.FixedZone("CustomTZ", 3600)

		WithTimezone(timezone)(c)

		assert.Equal(t, timezone, c.timeSource.Timezone, "timezone should be set correctly")
	})

	t.Run("WithPurgePercent", func(t *testing.T) {
		c := &cache{}
		percent := 25.0

		WithPurgePercent(percent)(c)

		assert.Equal(t, percent, c.purgePercent, "purgePercent should be set correctly")
	})

	t.Run("WithPurgeTimeout", func(t *testing.T) {
		c := &cache{}
		timeout := 5 * time.Minute

		WithPurgeTimeout(timeout)(c)

		assert.Equal(t, timeout, c.purgeTimeout, "purgeTimeout should be set correctly")
	})
}
