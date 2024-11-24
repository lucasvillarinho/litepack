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
	dbMocks "github.com/lucasvillarinho/litepack/database/mocks"
	"github.com/lucasvillarinho/litepack/internal/cron"
	logMocks "github.com/lucasvillarinho/litepack/internal/log/mocks"
)

func TestPurge_PurgeItens(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ctx := context.Background()

	t.Run("should purge and vacuum the database successfully", func(t *testing.T) {
		dbMock := dbMocks.NewDatabaseMock(t)

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
		dbMock := dbMocks.NewDatabaseMock(t)

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
		assert.Equal(
			t,
			"count entries: simulated failure",
			err.Error(),
			"Error should mention count entries failure",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})

	t.Run("should fail when Vacuum returns an error", func(t *testing.T) {
		dbMock := dbMocks.NewDatabaseMock(t)

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
		assert.Equal(
			t,
			"vacuum error: simulated failure",
			err.Error(),
			"Error should mention vacuuming cache failure",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})
}

func TestPurge_PurgeWithTransaction(t *testing.T) {
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

func TestPurge_purgeExpiredItensCache(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ctx := context.Background()
	tz := time.FixedZone("UTC", 0)
	timeMock := time.Date(2024, 11, 22, 12, 0, 0, 0, tz)
	loggerMock := logMocks.NewLoggerMock(t)
	ch := &cache{
		queries: queries.New(db),
		cron:    cron.New(tz),
		timeSource: timeSource{
			Timezone: tz,
			Now:      func() time.Time { return timeMock },
		},
		syncInterval: cron.EveryMinute,
		logger:       loggerMock,
	}

	t.Run("should clear expired itens from cache", func(t *testing.T) {
		sqlMock.ExpectExec(`DELETE FROM cache WHERE expires_at <= \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		ch.purgeExpiredItensCache(ctx)

		assert.NoError(t, err, "Expected no error while clearing expired itens")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should log error when clearing expired items", func(t *testing.T) {
		err := fmt.Errorf("unexpected error")
		errMock := fmt.Errorf("expired cache: %w", err)

		sqlMock.ExpectExec(`DELETE FROM cache WHERE expires_at <= \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(errMock)

		loggerMock.EXPECT().
			Error(mock.Anything, "deleting expired cache: expired cache: unexpected error")

		ch.purgeExpiredItensCache(ctx)

		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})
}

func TestPurgeItens(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ctx := context.Background()

	t.Run("should purge and vacuum the database successfully", func(t *testing.T) {
		dbMock := dbMocks.NewDatabaseMock(t)

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
		dbMock := dbMocks.NewDatabaseMock(t)

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
		assert.Equal(
			t,
			"count entries: simulated failure",
			err.Error(),
			"Error should mention count entries failure",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})

	t.Run("should fail when Vacuum returns an error", func(t *testing.T) {
		dbMock := dbMocks.NewDatabaseMock(t)

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
		assert.Equal(
			t,
			"vacuum error: simulated failure",
			err.Error(),
			"Error should mention vacuuming cache failure",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
		dbMock.AssertExpectations(t)
	})
}
