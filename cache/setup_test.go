package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lucasvillarinho/litepack/cache/queries"
	"github.com/lucasvillarinho/litepack/database/mocks"
)

func TestCache_Setup(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	dbMock := mocks.NewDatabaseMock(t)

	t.Run("should create the cache table successfully", func(t *testing.T) {
		sqlMock.ExpectExec("(?i)CREATE TABLE IF NOT EXISTS cache").
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock := mocks.NewDatabaseMock(t)
		dbMock.EXPECT().
			GetEngine(mock.Anything).
			Return(db)

		dbMock.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil)

		ch := &cache{
			queries:  queries.New(db),
			Database: dbMock,
		}

		err := ch.setupCacheTable(context.Background())

		assert.NoError(t, err, "Expected no error while creating the cache table")
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should return an error if table creation fails", func(t *testing.T) {
		sqlMock.ExpectExec("(?i)CREATE TABLE IF NOT EXISTS cache").
			WillReturnError(fmt.Errorf("mock create table error"))

		dbMock.EXPECT().
			GetEngine(mock.Anything).
			Return(db)

		ch := &cache{
			queries:  queries.New(db),
			Database: dbMock,
		}

		err := ch.setupCacheTable(context.Background())

		assert.Error(t, err, "Expected an error when table creation fails")
		assert.Equal(
			t,
			"creating table: mock create table error",
			err.Error(),
			"Expected error message to match",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should return an error if index creation fails", func(t *testing.T) {
		sqlMock.ExpectExec("(?i)CREATE TABLE IF NOT EXISTS cache").
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock := mocks.NewDatabaseMock(t)
		dbMock.EXPECT().
			GetEngine(mock.Anything).
			Return(db)

		dbMock.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(errors.New("unexpected error"))

		ch := &cache{
			queries:  queries.New(db),
			Database: dbMock,
		}

		err := ch.setupCacheTable(context.Background())

		assert.Error(t, err, "Expected an error when index creation fails")
		assert.Equal(
			t,
			"creating index: unexpected error",
			err.Error(),
			"Expected error message to match",
		)
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "Not all expectations were met")
	})
}
