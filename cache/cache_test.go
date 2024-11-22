package cache

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lucasvillarinho/litepack/cache/queries"
	"github.com/lucasvillarinho/litepack/database/mocks"
)

func TestCacheGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ch := &cache{
		timeSource: timeSource{
			Timezone: time.UTC,
		},
		queries: queries.New(db),
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

func TestCacheDel(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ch := &cache{
		timeSource: timeSource{
			Timezone: time.UTC,
		},
		queries: queries.New(db),
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
		assert.Equal(
			t,
			err.Error(),
			"deleting key: mock delete error",
			"Error message should match",
		)
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})
}

func TestSetupCache(t *testing.T) {

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	t.Run("should create the cache table successfully", func(t *testing.T) {
		sqlMock.ExpectExec("(?i)CREATE TABLE IF NOT EXISTS cache").
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock := mocks.NewDatabaseMock(t)
		dbMock.EXPECT().
			GetEngine(mock.Anything).
			Return(db)

		ch := &cache{
			queries:  queries.New(db),
			Database: dbMock,
		}

		err := ch.setupCache(context.Background())

		assert.NoError(t, err, "Expected no error while creating the cache table")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should return an error if table creation fails", func(t *testing.T) {
		sqlMock.ExpectExec("(?i)CREATE TABLE IF NOT EXISTS cache").
			WillReturnError(fmt.Errorf("mock create table error"))

		dbMock := mocks.NewDatabaseMock(t)
		dbMock.EXPECT().
			GetEngine(mock.Anything).
			Return(db)

		ch := &cache{
			queries:  queries.New(db),
			Database: dbMock,
		}

		err := ch.setupCache(context.Background())

		assert.Error(t, err, "Expected an error when table creation fails")
		assert.Equal(
			t,
			"error creating table: mock create table error",
			err.Error(),
			"Expected error message to match",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})
}

func TestPurgeItens(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ctx := context.Background()

	t.Run("should purge and vacuum the database successfully", func(t *testing.T) {
		dbMock := mocks.NewDatabaseMock(t)

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		sqlMock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnResult(sqlmock.NewResult(1, 20))
		sqlMock.ExpectCommit()

		dbMock.EXPECT().
			Vacuum(ctx, mock.Anything).Return(nil)
		dbMock.EXPECT().
			ExecWithTx(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, fn func(*sql.Tx) error) {

				tx, err := db.Begin()
				assert.NoError(t, err, "Expected no error while beginning transaction")

				err = fn(tx)
				assert.NoError(t, err, "Expected no error during transaction execution")

				err = tx.Commit()
				assert.NoError(t, err, "Expected no error while committing transaction")
			}).
			Return(nil)

		ch := &cache{
			queries:      queries.New(db),
			purgePercent: 0.2,
			Database:     dbMock,
		}

		err := ch.PurgeItens(context.Background())

		assert.NoError(t, err, "Expected no error while purging and vacuuming the database")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})

	t.Run("should fail when PurgeEntriesByPercentage returns an error", func(t *testing.T) {
		dbMock := mocks.NewDatabaseMock(t)

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnError(fmt.Errorf("database error")) // Simula falha no CountEntries
		sqlMock.ExpectRollback()

		dbMock.EXPECT().
			ExecWithTx(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, fn func(*sql.Tx) error) {
				tx, err := db.Begin()
				assert.NoError(t, err, "Expected no error while beginning transaction")

				err = fn(tx)
				assert.Error(t, err, "Expected error during transaction execution")

				err = tx.Rollback()
				assert.NoError(t, err, "Expected no error while rolling back transaction")
			}).
			Return(fmt.Errorf("count entries: simulated failure"))

		ch := &cache{
			queries:      queries.New(db),
			purgePercent: 0.2,
			Database:     dbMock,
		}

		err := ch.PurgeItens(context.Background())

		assert.Error(t, err, "Expected error while purging items")
		assert.Equal(t, "count entries: simulated failure", err.Error(), "Error should mention count entries failure")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})

	t.Run("should fail when Vacuum returns an error", func(t *testing.T) {
		dbMock := mocks.NewDatabaseMock(t)

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		sqlMock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnResult(sqlmock.NewResult(1, 20))
		sqlMock.ExpectRollback()

		dbMock.EXPECT().
			ExecWithTx(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, fn func(*sql.Tx) error) {
				tx, err := db.Begin()
				assert.NoError(t, err, "Expected no error while beginning transaction")

				err = fn(tx)
				assert.Error(t, err, "Expected error during transaction execution")

				err = tx.Rollback()
				assert.NoError(t, err, "Expected no error while rolling back transaction")
			}).
			Return(fmt.Errorf("vacuum error: simulated failure"))

		dbMock.EXPECT().
			Vacuum(ctx, mock.Anything).
			Return(fmt.Errorf("vacuum error: simulated failure"))

		ch := &cache{
			queries:      queries.New(db),
			purgePercent: 0.2,
			Database:     dbMock,
		}

		err := ch.PurgeItens(context.Background())

		assert.Error(t, err, "Expected error while vacuuming")
		assert.Equal(t, "vacuum error: simulated failure", err.Error(), "Error should mention vacuuming cache failure")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
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

		err = ch.purgeEntriesByPercentage(context.Background(), tx, 0.2)

		assert.NoError(t, err, "Expected no error while purging entries")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("Should return error if percentage is invalid", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		ch := &cache{
			queries: queries.New(tx),
		}

		err = ch.purgeEntriesByPercentage(context.Background(), tx, 1.2)

		assert.Error(t, err, "Expected an error for invalid percentage")
		assert.Equal(t, "invalid percentage: 1.200000", err.Error(), "Error message should match")
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

		err = ch.purgeEntriesByPercentage(context.Background(), tx, 0.2)

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

		err = ch.purgeEntriesByPercentage(context.Background(), tx, 0.2)

		assert.Error(t, err, "Expected an error for failing SELECT query")
		assert.Equal(
			t,
			"count entries: mock select error",
			err.Error(),
			"Error message should match",
		)
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

		err = ch.purgeEntriesByPercentage(context.Background(), tx, 0.2)
		assert.Error(t, err, "Expected an error for failing DELETE query")
		assert.Equal(
			t,
			"delete entries: mock delete error",
			err.Error(),
			"Error message should match",
		)
	})
}

func TestSetCache(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	tz := time.FixedZone("UTC", 0)
	fixedTime := time.Date(2024, 11, 22, 12, 0, 0, 0, tz)
	ctx := context.Background()

	ch := &cache{
		queries: queries.New(db),
		timeSource: timeSource{
			Timezone: tz,
			Now:      func() time.Time { return fixedTime },
		},
		purgePercent: 0.2,
	}

	t.Run("should successfully set a cache item", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := 1 * time.Hour

		expectedExpiresAt := fixedTime.Add(ttl)
		expectedLastAccessedAt := fixedTime

		sqlMock.ExpectExec(`INSERT INTO cache \(key, value, expires_at, last_accessed_at\) VALUES \(\?, \?, \?, \?\) ON CONFLICT \(key\) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at`).
			WithArgs(
				key,
				value,
				expectedExpiresAt,
				expectedLastAccessedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := ch.Set(context.Background(), key, value, ttl)

		assert.NoError(t, err, "Expected no error when setting cache")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should retry the set operation if the database is full", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := 1 * time.Hour

		expectedExpiresAt := fixedTime.Add(ttl)
		expectedLastAccessedAt := fixedTime
		dbMock := mocks.NewDatabaseMock(t)
		ch.Database = dbMock

		// First attempt to set the cache item
		sqlMock.ExpectExec(`INSERT INTO cache \(key, value, expires_at, last_accessed_at\) VALUES \(\?, \?, \?, \?\) ON CONFLICT \(key\) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at`).
			WithArgs(
				key,
				value,
				expectedExpiresAt,
				expectedLastAccessedAt,
			).
			WillReturnError(fmt.Errorf("database or disk is full"))

		// Purge the cache and retry the set operation
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		sqlMock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnResult(sqlmock.NewResult(1, 20))
		sqlMock.ExpectCommit()

		dbMock.EXPECT().
			Vacuum(ctx, mock.Anything).
			Return(nil).
			Times(1)
		dbMock.EXPECT().
			ExecWithTx(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, fn func(*sql.Tx) error) {
				tx, err := db.Begin()
				assert.NoError(t, err, "Expected no error while beginning transaction")

				err = fn(tx)
				assert.NoError(t, err, "Expected no error during transaction execution")

				err = tx.Commit()
				assert.NoError(t, err, "Expected no error while committing transaction")
			}).
			Return(nil).
			Times(1)

		// Retry the set operation
		sqlMock.ExpectExec(`INSERT INTO cache \(key, value, expires_at, last_accessed_at\) VALUES \(\?, \?, \?, \?\) ON CONFLICT \(key\) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at`).
			WithArgs(
				key,
				value,
				expectedExpiresAt,
				expectedLastAccessedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := ch.Set(context.Background(), key, value, ttl)
		assert.NoError(t, err, "Expected no error when setting cache")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should return error if the set operation fails after retrying", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := 1 * time.Hour

		expectedExpiresAt := fixedTime.Add(ttl)
		expectedLastAccessedAt := fixedTime
		dbMock := mocks.NewDatabaseMock(t)
		ch.Database = dbMock

		// First attempt to set the cache item
		sqlMock.ExpectExec(`INSERT INTO cache \(key, value, expires_at, last_accessed_at\) VALUES \(\?, \?, \?, \?\) ON CONFLICT \(key\) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at`).
			WithArgs(
				key,
				value,
				expectedExpiresAt,
				expectedLastAccessedAt,
			).
			WillReturnError(fmt.Errorf("database or disk is full"))

		// Purge the cache and retry the set operation
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM cache`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(100))
		sqlMock.ExpectExec(`DELETE FROM cache WHERE key IN \( SELECT key FROM cache ORDER BY last_accessed_at ASC LIMIT \? \)`).
			WithArgs(20).
			WillReturnResult(sqlmock.NewResult(1, 20))
		sqlMock.ExpectCommit()

		dbMock.EXPECT().
			Vacuum(ctx, mock.Anything).
			Return(nil).
			Times(1)
		dbMock.EXPECT().
			ExecWithTx(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, fn func(*sql.Tx) error) {
				tx, err := db.Begin()
				assert.NoError(t, err, "Expected no error while beginning transaction")

				err = fn(tx)
				assert.NoError(t, err, "Expected no error during transaction execution")

				err = tx.Commit()
				assert.NoError(t, err, "Expected no error while committing transaction")
			}).
			Return(nil).
			Times(1)

		// Retry the set operation
		sqlMock.ExpectExec(`INSERT INTO cache \(key, value, expires_at, last_accessed_at\) VALUES \(\?, \?, \?, \?\) ON CONFLICT \(key\) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, last_accessed_at = excluded.last_accessed_at`).
			WithArgs(
				key,
				value,
				expectedExpiresAt,
				expectedLastAccessedAt,
			).
			WillReturnError(fmt.Errorf("database or disk is full"))

		err := ch.Set(context.Background(), key, value, ttl)
		assert.Error(t, err, "Expected an error when setting cache")
		assert.Equal(t, "error setting cache: database or disk is full", err.Error(), "Error message should match")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

}
