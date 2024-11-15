package cache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/database/drivers"
)

func TestSetWalMode(t *testing.T) {

	t.Run("should enable WAL mode successfully", func(t *testing.T) {
		mock := &drivers.Mock{}
		ch := &cache{
			engine: mock,
		}

		err := setWalMode(ch)

		assert.NoError(t, err, "Expected no error when enabling WAL mode")
		assert.Equal(t, "PRAGMA journal_mode=WAL;", mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when enabling WAL mode fails", func(t *testing.T) {
		mock := &drivers.Mock{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := setWalMode(ch)

		assert.Error(t, err, "Expected an error when enabling WAL mode")
		assert.EqualError(t, err, "enabling WAL mode: mock error", "Expected error message to match")
	})
}

func TestSetSynchronousMode(t *testing.T) {

	t.Run("should set synchronous mode successfully", func(t *testing.T) {
		mock := &drivers.Mock{}
		ch := &cache{
			engine: mock,
		}

		err := setSynchronousMode(ch)

		assert.NoError(t, err, "Expected no error when setting synchronous mode")
		assert.Equal(t, "PRAGMA synchronous = NORMAL;", mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when setting synchronous mode fails", func(t *testing.T) {
		mock := &drivers.Mock{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := setSynchronousMode(ch)

		assert.Error(t, err, "Expected an error when setting synchronous mode")
		assert.EqualError(t, err, "setting synchronous mode: mock error", "Expected error message to match")
	})
}

func TestCreateIndex(t *testing.T) {

	t.Run("should create index successfully", func(t *testing.T) {
		mock := &drivers.Mock{}
		ch := &cache{
			engine: mock,
		}

		err := createIndex(ch)

		assert.NoError(t, err, "Expected no error when creating index")
		assert.Equal(t, "CREATE INDEX IF NOT EXISTS idx_key ON cache (key);", mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when creating index fails", func(t *testing.T) {
		mock := &drivers.Mock{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := createIndex(ch)

		assert.Error(t, err, "Expected an error when creating index")
		assert.EqualError(t, err, "creating index: mock error", "Expected error message to match")
	})
}

func TestSetCacheSize(t *testing.T) {
	t.Run("should set cache size successfully", func(t *testing.T) {
		mock := &drivers.Mock{}
		ch := &cache{
			engine:    mock,
			cacheSize: 128 * 1024 * 1024, // 128 MB
		}

		err := setCacheSize(ch)

		expectedQuery := "PRAGMA cache_size = 32768;" // 128 MB / 4096 bytes
		assert.NoError(t, err, "Expected no error when setting cache size")
		assert.Equal(t, expectedQuery, mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when setting cache size fails", func(t *testing.T) {
		mock := &drivers.Mock{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine:    mock,
			cacheSize: 128 * 1024 * 1024, // 128 MB
		}

		err := setCacheSize(ch)

		assert.Error(t, err, "Expected an error when setting cache size")
		assert.EqualError(t, err, "setting cache size: mock error", "Expected error message to match")
	})

}
