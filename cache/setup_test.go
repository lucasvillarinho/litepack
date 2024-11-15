package cache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/database/drivers"
	mocks "github.com/lucasvillarinho/litepack/database/drivers/mocks"
	"github.com/lucasvillarinho/litepack/internal/helpers"
)

func TestSetWalMode(t *testing.T) {
	t.Run("should enable WAL mode successfully", func(t *testing.T) {
		mock := &mocks.MockEngine{}
		ch := &cache{
			engine: mock,
		}

		err := setWalMode(ch)

		assert.NoError(t, err, "Expected no error when enabling WAL mode")
		assert.Equal(t, "PRAGMA journal_mode=WAL;", mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when enabling WAL mode fails", func(t *testing.T) {
		mock := &mocks.MockEngine{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := setWalMode(ch)

		assert.Error(t, err, "Expected an error when enabling WAL mode")
		assert.EqualError(
			t,
			err,
			"enabling WAL mode: mock error",
			"Expected error message to match",
		)
	})
}

func TestSetSynchronousMode(t *testing.T) {
	t.Run("should set synchronous mode successfully", func(t *testing.T) {
		mock := &mocks.MockEngine{}
		ch := &cache{
			engine: mock,
		}

		err := setSynchronousMode(ch)

		assert.NoError(t, err, "Expected no error when setting synchronous mode")
		assert.Equal(
			t,
			"PRAGMA synchronous = NORMAL;",
			mock.ExecutedQuery,
			"Expected query to match",
		)
	})

	t.Run("should return an error when setting synchronous mode fails", func(t *testing.T) {
		mock := &mocks.MockEngine{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := setSynchronousMode(ch)

		assert.Error(t, err, "Expected an error when setting synchronous mode")
		assert.EqualError(
			t,
			err,
			"setting synchronous mode: mock error",
			"Expected error message to match",
		)
	})
}

func TestCreateIndex(t *testing.T) {
	t.Run("should create index successfully", func(t *testing.T) {
		mock := &mocks.MockEngine{}
		ch := &cache{
			engine: mock,
		}

		err := createIndex(ch)

		assert.NoError(t, err, "Expected no error when creating index")
		assert.Equal(
			t,
			"CREATE INDEX IF NOT EXISTS idx_key ON cache (key);",
			mock.ExecutedQuery,
			"Expected query to match",
		)
	})

	t.Run("should return an error when creating index fails", func(t *testing.T) {
		mock := &mocks.MockEngine{
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
		mock := &mocks.MockEngine{}
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
		mock := &mocks.MockEngine{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine:    mock,
			cacheSize: 128 * 1024 * 1024, // 128 MB
		}

		err := setCacheSize(ch)

		assert.Error(t, err, "Expected an error when setting cache size")
		assert.EqualError(
			t,
			err,
			"setting cache size: mock error",
			"Expected error message to match",
		)
	})
}

func TestCreateCacheTable(t *testing.T) {
	t.Run("should create cache table successfully", func(t *testing.T) {
		mock := &mocks.MockEngine{}
		ch := &cache{
			engine: mock,
		}

		err := createCacheTable(ch)

		expectedQuery := `
    CREATE TABLE IF NOT EXISTS cache (
        key TEXT PRIMARY KEY,
        value BLOB,
        expires_at TIMESTAMP,
	    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
		assert.NoError(t, err)
		assert.Equal(t, helpers.NormalizeQuery(expectedQuery), helpers.NormalizeQuery(mock.ExecutedQuery))
	})

	t.Run("should return an error when creating cache table fails", func(t *testing.T) {
		mock := &mocks.MockEngine{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := createCacheTable(ch)

		expectedQuery := `
    CREATE TABLE IF NOT EXISTS cache (
        key TEXT PRIMARY KEY,
        value BLOB,
        expires_at TIMESTAMP,
        last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
		assert.Error(t, err)
		assert.Equal(t, helpers.NormalizeQuery(expectedQuery), helpers.NormalizeQuery(mock.ExecutedQuery))
		assert.EqualError(t, err, "creating table: mock error")
	})
}

func TestSetDriver(t *testing.T) {
	t.Run("should set the driver successfully", func(t *testing.T) {
		mockDriver := &mocks.MockEngine{}
		mockFactory := &mocks.MockDriverFactory{
			MockDriver: mockDriver,
		}

		ch := &cache{
			drive: drivers.DriverMattn,
			dsn:   "mock_dsn",
		}

		err := setDriver(ch, mockFactory)

		assert.NoError(t, err)
		assert.Equal(t, mockDriver, ch.engine)
	})

	t.Run("should return an error when getting the driver fails", func(t *testing.T) {
		mockFactory := &mocks.MockDriverFactory{
			Error: fmt.Errorf("mock error"),
		}

		ch := &cache{
			drive: drivers.DriverMattn,
			dsn:   "mock_dsn",
		}

		err := setDriver(ch, mockFactory)

		assert.Error(t, err)
		assert.EqualError(t, err, "error getting driver: mock error")
	})
}
