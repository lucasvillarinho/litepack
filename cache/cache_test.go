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
		key := "existing_key"

		mock.ExpectQuery(`SELECT value FROM cache WHERE`).
			WithArgs(key, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).
				AddRow(expectedValue))
		mock.ExpectExec(`UPDATE cache SET last_accessed_at = \? WHERE key = \?`).
			WithArgs(sqlmock.AnyArg(), key).
			WillReturnResult(sqlmock.NewResult(1, 1))

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

	t.Run("Should return nil if UPDATE query fails", func(t *testing.T) {
		expectedValue := "cached_data"
		key := "existing_key"

		mock.ExpectQuery(`SELECT value FROM cache WHERE`).
			WithArgs(key, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).
				AddRow(expectedValue))
		mock.ExpectExec(`UPDATE cache SET last_accessed_at = \? WHERE key = \?`).
			WithArgs(sqlmock.AnyArg(), key).
			WillReturnError(sql.ErrConnDone)

		value, err := ch.Get(context.Background(), key)

		assert.Equal(t, []byte(expectedValue), value, "Expected cached value to match")
		assert.Nil(t, err, "Expected no error for failing UPDATE query")
	})
}

func TestSetupDatabase(t *testing.T) {
	ctx := context.Background()

	t.Run("should set up the database successfully", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA synchronous = NORMAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA max_page_count = 32768;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA page_size = 4096;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA cache_size = 32768;").
			Return(nil, nil)

		c := &cache{
			dsn:       "mock_dsn",
			engine:    driverMock,
			dbSize:    128 * 1024 * 1024,
			pageSize:  4096,
			cacheSize: 128 * 1024 * 1024,
		}

		err := c.setupDatabase(ctx)

		assert.NoError(t, err, "Expected no error during database setup")
		driverMock.AssertExpectations(t)
	})

	t.Run("should return an error when enabling WAL mode fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, fmt.Errorf("mock error enabling WAL mode"))

		c := &cache{
			dsn:    "mock_dsn",
			engine: driverMock,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when enabling WAL mode fails")
		assert.Equal(t, "enabling WAL mode: mock error enabling WAL mode", err.Error())
		driverMock.AssertExpectations(t)
	})

	t.Run("should return an error when setting synchronous mode fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, nil)

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA synchronous = NORMAL;").
			Return(nil, fmt.Errorf("mock error setting synchronous mode"))

		c := &cache{
			dsn:    "mock_dsn",
			engine: driverMock,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting synchronous mode fails")
		assert.Equal(t, "setting synchronous mode: mock error setting synchronous mode", err.Error())
		driverMock.AssertExpectations(t)
	})

	t.Run("should return error when setting maximum page count fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		pageSize := 4096
		dbSize := 128 * 1024 * 1024
		maxPageCount := dbSize / pageSize

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA synchronous = NORMAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d;", maxPageCount)).
			Return(nil, fmt.Errorf("mock error"))

		c := &cache{
			dsn:      "mock_dsn",
			engine:   driverMock,
			dbSize:   dbSize,
			pageSize: pageSize,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting max page count fails")
		assert.Equal(t, "setting max page count: mock error", err.Error())
		driverMock.AssertExpectations(t)
	})

	t.Run("should return error when setting page size fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		pageSize := 4096
		cacheSize := 128 * 1024 * 1024
		dbSize := 128 * 1024 * 1024
		expectedMaxPageCount := dbSize / pageSize

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA synchronous = NORMAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d;", expectedMaxPageCount)).
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA page_size = %d;", pageSize)).
			Return(nil, fmt.Errorf("mock error"))

		c := &cache{
			dsn:       "mock_dsn",
			engine:    driverMock,
			dbSize:    dbSize,
			pageSize:  pageSize,
			cacheSize: cacheSize,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting page size fails")
		assert.Equal(t, "setting page size: mock error", err.Error())
		driverMock.AssertExpectations(t)
	})

	t.Run("should return error when setting cache size fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		pageSize := 4096
		cacheSize := 128 * 1024 * 1024
		dbSize := 128 * 1024 * 1024
		expectedCacheSize := cacheSize / pageSize
		expectedMaxPageCount := dbSize / pageSize

		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA journal_mode=WAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, "PRAGMA synchronous = NORMAL;").
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d;", expectedMaxPageCount)).
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA page_size = %d;", pageSize)).
			Return(nil, nil)
		driverMock.EXPECT().
			ExecContext(ctx, fmt.Sprintf("PRAGMA cache_size = %d;", expectedCacheSize)).
			Return(nil, fmt.Errorf("mock error"))

		c := &cache{
			dsn:       "mock_dsn",
			engine:    driverMock,
			pageSize:  pageSize,
			cacheSize: cacheSize,
			dbSize:    dbSize,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting cache size fails")
		assert.Equal(t, "setting cache size: mock error", err.Error())
		driverMock.AssertExpectations(t)
	})
}

func TestSetupEngine(t *testing.T) {
	t.Run("should set up the engine successfully", func(t *testing.T) {

		c := &cache{
			dsn: "mock_dsn",
		}

		err := c.setupEngine(context.Background())

		assert.NoError(t, err, "Expected no error when setting up the engine")
		assert.NotNil(t, c.engine, "Engine should be initialized")
		assert.NotNil(t, c.queries, "Queries should be initialized")
	})
}

func TestCacheDel(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ch := &cache{
		timezone: time.UTC,
		queries:  queries.New(db),
	}

	t.Run("Should delete the key successfully", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM cache WHERE key = ?`).
			WithArgs("existing_key").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := ch.Del(context.Background(), "existing_key")

		assert.NoError(t, err, "Expected no error while deleting the key")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return nil if the key does not exist", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM cache WHERE key = ?`).
			WithArgs("non_existing_key").
			WillReturnResult(sqlmock.NewResult(1, 0))

		err := ch.Del(context.Background(), "non_existing_key")

		assert.NoError(t, err, "Expected no error for non-existing key")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return error if DELETE query fails", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM cache WHERE key = ?`).
			WithArgs("error_key").
			WillReturnError(fmt.Errorf("mock delete error"))

		err := ch.Del(context.Background(), "error_key")

		assert.Error(t, err, "Expected an error for failing DELETE query")
		assert.Equal(t, err.Error(), "deleting key: mock delete error", "Error message should match")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})
}

func TestCachePurgeWithTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	t.Run("Should purge the correct percentage of entries", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(100))
		mock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnResult(sqlmock.NewResult(1, 20))

		ch := &cache{
			queries: queries.New(tx),
		}

		err = ch.PurgeWithTransaction(context.Background(), 0.2, tx)

		assert.NoError(t, err, "Expected no error while purging entries")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return nil if there are no entries to delete", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		ch := &cache{
			queries: queries.New(tx),
		}

		err = ch.PurgeWithTransaction(context.Background(), 0.2, tx)

		assert.NoError(t, err, "Expected no error while purging entries")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return error if SELECT query fails", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnError(fmt.Errorf("mock select error"))

		ch := &cache{
			queries: queries.New(tx),
		}

		err = ch.PurgeWithTransaction(context.Background(), 0.2, tx)

		assert.Error(t, err, "Expected an error for failing SELECT query")
		assert.Equal(t, err.Error(), "error to count entries: mock select error", "Error message should match")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return error if DELETE query fails", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(100))
		mock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnError(fmt.Errorf("mock delete error"))

		ch := &cache{
			queries: queries.New(tx),
		}

		err = ch.PurgeWithTransaction(context.Background(), 0.2, tx)

		assert.Error(t, err, "Expected an error for failing DELETE query")
		assert.Equal(t, err.Error(), "error to delete entries: mock delete error", "Error message should match")
	})

}
