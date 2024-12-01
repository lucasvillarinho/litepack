package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	lPCache "github.com/lucasvillarinho/litepack/cache"
)

func TestCache(t *testing.T) {
	ctx := context.Background()

	lCache, err := lPCache.NewCache(ctx)
	if err != nil {
		panic(err)
	}
	defer lCache.Destroy(ctx)

	t.Run("Should successfully set cache entry ", func(t *testing.T) {
		defer lCache.Del(ctx, "key")

		err := lCache.Set(ctx, "key", "test", 10*time.Second)

		assert.Nil(t, err, "Expected to set cache entry without error, but got: %v", err)
	})

	t.Run("Should successfully get cache entry ", func(t *testing.T) {
		defer lCache.Del(ctx, "key")

		_ = lCache.Set(ctx, "key", "test", 10*time.Second)

		value, err := lCache.Get(ctx, "key")

		assert.Nil(t, err, "Expected to get cache entry without error, but got: %v", err)
		assert.Equal(
			t,
			"test",
			string(value),
			"Expected to get cache entry with value 'test', but got: %v",
			string(value),
		)
	})

	t.Run("Should successfully delete cache entry ", func(t *testing.T) {
		_ = lCache.Set(ctx, "key", "test", 10*time.Second)

		err := lCache.Del(ctx, "key")
		if err != nil {
			t.Errorf("Expected to delete cache entry without error, but got: %v", err)
		}

		value, err := lCache.Get(ctx, "key")

		assert.Error(t, err, "Expected to get error when trying to get deleted cache entry")
		assert.Equal(t, lPCache.ErrKeyNotFound, err, "Expected to get error 'key not found', but got: %v", err)
		assert.Emptyf(t, value, "Expected to get empty cache entry, but got: %v", value)
	})
}
