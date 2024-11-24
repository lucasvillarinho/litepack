package log

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	mdb "github.com/lucasvillarinho/litepack/database/mocks"
	"github.com/lucasvillarinho/litepack/internal/log/queries"
)

func TestLoggerError(t *testing.T) {
	t.Run("should log an error successfully", func(t *testing.T) {
		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		sqlMock.ExpectExec(`INSERT INTO log \(level, message\) VALUES \(\?, \?\)`).
			WithArgs("ERROR", "test error").
			WillReturnResult(sqlmock.NewResult(1, 1))

		ctx := context.Background()
		lg := &logger{
			queries: queries.New(db),
		}

		lg.Error(ctx, errors.New("test error").Error())

		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestNewLogger(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	t.Run("should create the logger successfully", func(t *testing.T) {
		sqlMock.ExpectExec("CREATE TABLE IF NOT EXISTS log").
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockDB := mdb.NewDatabaseMock(t)
		mockDB.EXPECT().
			GetEngine(context.Background()).
			Return(db)

		lg, err := NewLogger(ctx, mockDB)

		assert.NoError(t, err, "Expected no error while creating the logger")
		assert.NotNil(t, lg, "Expected a valid logger instance")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all SQL expectations were met")
	})

	t.Run("should return an error if table creation fails", func(t *testing.T) {
		sqlMock.ExpectExec("CREATE TABLE IF NOT EXISTS log").
			WillReturnError(fmt.Errorf("mock create table error"))

		mockDB := mdb.NewDatabaseMock(t)
		mockDB.EXPECT().
			GetEngine(context.Background()).
			Return(db)

		ctx := context.Background()
		lg, err := NewLogger(ctx, mockDB)

		assert.Error(t, err, "Expected an error when table creation fails")
		assert.Nil(t, lg, "Expected logger instance to be nil on error")
		assert.Contains(
			t,
			err.Error(),
			"failed to create log table",
			"Expected error message to match",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all SQL expectations were met")
	})
}
