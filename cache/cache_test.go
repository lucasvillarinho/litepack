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
		driverMock := mocks.NewDriverMock(t)

		// Configurando as expectativas do mock
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
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    driverMock,
			dbSize:    128 * 1024 * 1024, // 128 MB
			pageSize:  4096,              // 4 KB
			cacheSize: 128 * 1024 * 1024, // 128 MB
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
			drive:  drivers.DriverMattn,
			dsn:    "mock_dsn",
			engine: driverMock,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when enabling WAL mode fails")
		assert.Contains(t, err.Error(), "mock error enabling WAL mode")
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
			drive:  drivers.DriverMattn,
			dsn:    "mock_dsn",
			engine: driverMock,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting synchronous mode fails")
		assert.Contains(t, err.Error(), "mock error setting synchronous mode")
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
			drive:    drivers.DriverMattn,
			dsn:      "mock_dsn",
			engine:   driverMock,
			dbSize:   dbSize,
			pageSize: pageSize,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting max page count fails")
		assert.Contains(t, err.Error(), "setting max page count: mock error")
		driverMock.AssertExpectations(t)
	})

	t.Run("should return error when setting page size fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		pageSize := 4096
		cacheSize := 128 * 1024 * 1024
		dbSize := 128 * 1024 * 1024 // Define o tamanho do banco corretamente
		expectedMaxPageCount := dbSize / pageSize

		// Configura as expectativas do mock
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
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    driverMock,
			dbSize:    dbSize,    // Adiciona o dbSize correto
			pageSize:  pageSize,  // Define o pageSize
			cacheSize: cacheSize, // Define o cacheSize
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting page size fails")
		assert.Contains(t, err.Error(), "setting page size: mock error")
		driverMock.AssertExpectations(t)
	})

	t.Run("should return error when setting cache size fails", func(t *testing.T) {
		driverMock := mocks.NewDriverMock(t)

		pageSize := 4096
		cacheSize := 128 * 1024 * 1024
		dbSize := 128 * 1024 * 1024 // Define o tamanho do banco
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
			drive:     drivers.DriverMattn,
			dsn:       "mock_dsn",
			engine:    driverMock,
			pageSize:  pageSize,
			cacheSize: cacheSize,
			dbSize:    dbSize,
		}

		err := c.setupDatabase(ctx)

		assert.Error(t, err, "Expected an error when setting cache size fails")
		assert.Contains(t, err.Error(), "setting cache size: mock error")
		driverMock.AssertExpectations(t)
	})
}
