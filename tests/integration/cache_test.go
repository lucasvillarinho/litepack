package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/cache"
)

func TestCache(t *testing.T) {
	liteCache, err := cache.NewCache("tests")
	if err != nil {
		panic(err)
	}
	defer liteCache.Destroy()

	t.Run("Should successfully set cache entry ", func(t *testing.T) {
		defer liteCache.Del("key")

		err := liteCache.Set("key", []byte("test"), 10*time.Second)

		assert.Nil(t, err, "Expected to set cache entry without error, but got: %v", err)
	})

	t.Run("Should successfully get cache entry ", func(t *testing.T) {
		defer liteCache.Del("key")

		_ = liteCache.Set("key", []byte("test"), 10*time.Second)

		value, err := liteCache.Get("key")

		assert.Nil(t, err, "Expected to get cache entry without error, but got: %v", err)
		assert.Equal(t, "test", string(value), "Expected to get cache entry with value 'test', but got: %v", string(value))
	})

	t.Run("Should successfully delete cache entry ", func(t *testing.T) {
		_ = liteCache.Set("key", []byte("test"), 10*time.Second)

		err := liteCache.Del("key")
		if err != nil {
			t.Errorf("Expected to delete cache entry without error, but got: %v", err)
		}

		value, err := liteCache.Get("key")

		assert.Nil(t, err, "Expected to get cache entry without error, but got: %v", err)
		assert.Nil(t, value, "Expected to get cache entry with value nil, but got: %v", value)
	})
}
