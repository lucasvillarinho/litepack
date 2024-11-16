package cache

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/cache/queries"
	"github.com/lucasvillarinho/litepack/database/drivers"
	mocks "github.com/lucasvillarinho/litepack/database/drivers/mocks"
)

func TestCacheGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ch := &cache{
		timezone: time.UTC,
		queries:  queries.New(db),
	}

	t.Run("Should return value if key exists and is not expired", func(t *testing.T) {
		expectedValue := "cached_data"

		mock.ExpectQuery(`SELECT value FROM cache WHERE`).
			WithArgs("existing_key", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(expectedValue))

		value, err := ch.Get(context.Background(), "existing_key")

		assert.NoError(t, err, "Expected no error, but got: %v", err)
		assert.Equal(t, []byte(expectedValue), value, "Expected cached value to match")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return nil if key does not exist (sql.ErrNoRows)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT value FROM cache WHERE`).
			WithArgs("non_existing_key", sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		value, err := ch.Get(context.Background(), "non_existing_key")

		assert.NoError(t, err, "Expected no error for non-existing key")
		assert.Nil(t, value, "Expected nil value for non-existing key")
	})

	t.Run("Should return error if query fails", func(t *testing.T) {
		mock.ExpectQuery(`SELECT value FROM cache WHERE`).
			WithArgs("error_key", sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)

		value, err := ch.Get(context.Background(), "error_key")

		assert.Error(t, err, "Expected error for failing query")
		assert.Nil(t, value, "Expected nil value for failing query")
	})
}

func TestSetupDatabase(t *testing.T) {
	ctx := context.Background()

	t.Run("should set up the database successfully", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{}

		c := &cache{
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    mockEngine,
			dbSize:    128 * 1024 * 1024, // 128 MB
			pageSize:  4096,              // 4 KB
			cacheSize: 128 * 1024 * 1024, // 128 MB
		}

		err := c.setupDatabase(ctx)

		assert.NoError(t, err, "Expected no error during database setup")
		assert.Equal(
			t,
			"PRAGMA journal_mode=WAL;",
			mockEngine.ExecutedQueries[0],
			"Expected journal_mode to be set to WAL",
		)
		assert.Equal(
			t,
			"PRAGMA synchronous = NORMAL;",
			mockEngine.ExecutedQueries[1],
			"Expected synchronous mode to be set to NORMAL",
		)
		assert.Equal(
			t,
			"PRAGMA max_page_count = 32768;",
			mockEngine.ExecutedQueries[2],
			"Expected max page count query to match",
		)
		assert.Equal(
			t,
			"PRAGMA page_size = 4096;",
			mockEngine.ExecutedQueries[3],
			"Expected page size query to match",
		)
		assert.Equal(
			t,
			"PRAGMA cache_size = 32768;",
			mockEngine.ExecutedQueries[4],
			"Expected cache size query to match",
		)
	})

	t.Run("should return an error when enabling WAL mode fails", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{
			QueryErrors: map[string]error{
				"PRAGMA journal_mode=WAL;": fmt.Errorf("mock error enabling WAL mode"),
			},
		}

		c := &cache{
			drive:  drivers.DriverMattn,
			dsn:    "mock_dsn",
			engine: mockEngine,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when enabling WAL mode fails")
		assert.Equal(
			t,
			"PRAGMA journal_mode=WAL;",
			mockEngine.ExecutedQueries[0],
			"Expected journal_mode to be set to WAL",
		)
		assert.Contains(
			t,
			err.Error(),
			"enabling WAL mode: mock error enabling WAL mode",
			"Error message should match",
		)
	})

	t.Run("should return an error when setting synchronous mode fails", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{
			QueryErrors: map[string]error{
				"PRAGMA synchronous = NORMAL;": fmt.Errorf("mock error setting synchronous mode"),
			},
		}

		c := &cache{
			drive:  drivers.DriverMattn,
			dsn:    "mock_dsn",
			engine: mockEngine,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting synchronous mode fails")
		assert.Equal(
			t,
			"PRAGMA journal_mode=WAL;",
			mockEngine.ExecutedQueries[0],
			"Expected journal_mode to be set to WAL",
		)
		assert.Contains(
			t,
			err.Error(),
			"setting synchronous mode: mock error setting synchronous mode",
			"Error message should match",
		)
	})

	t.Run("should return an error when setting max page count fails", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{
			QueryErrors: map[string]error{
				"PRAGMA max_page_count = 32768;": fmt.Errorf("mock error setting max page count"),
			},
		}

		c := &cache{
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    mockEngine,
			dbSize:    128 * 1024 * 1024, // 128 MB
			cacheSize: 128 * 1024 * 1024, // 128 MB
			pageSize:  4096,              // 4 KB
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting max page count fails")
		assert.Equal(
			t,
			"PRAGMA max_page_count = 32768;",
			mockEngine.ExecutedQueries[2],
			"Expected max page count query to match",
		)
		assert.Contains(
			t,
			err.Error(),
			"setting max page count: mock error setting max page count",
			"Error message should match",
		)
	})

	t.Run("should return an error when setting page size fails", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{
			QueryErrors: map[string]error{
				"PRAGMA page_size = 4096;": fmt.Errorf("mock error setting page size"),
			},
		}

		c := &cache{
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    mockEngine,
			dbSize:    128 * 1024 * 1024, // 128 MB
			cacheSize: 128 * 1024 * 1024, // 128 MB
			pageSize:  4096,              // 4 KB
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting page size fails")
		assert.Equal(
			t,
			"PRAGMA page_size = 4096;",
			mockEngine.ExecutedQueries[3],
			"Expected page size query to match",
		)
		assert.Contains(
			t,
			err.Error(),
			"setting page size: mock error setting page size",
			"Error message should match",
		)
	})

	t.Run("should return an error when setting cache size fails", func(t *testing.T) {
		mockEngine := &mocks.MockEngine{
			QueryErrors: map[string]error{
				"PRAGMA cache_size = 32768;": fmt.Errorf("mock error setting cache size"),
			},
		}

		c := &cache{
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    mockEngine,
			dbSize:    128 * 1024 * 1024, // 128 MB
			cacheSize: 128 * 1024 * 1024, // 128 MB
			pageSize:  4096,              // 4 KB
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting cache size fails")
		assert.Equal(
			t,
			"PRAGMA cache_size = 32768;",
			mockEngine.ExecutedQueries[4],
			"Expected cache size query to match",
		)
		assert.Contains(
			t,
			err.Error(),
			"setting cache size: mock error setting cache size",
			"Error message should match",
		)
	})
}
