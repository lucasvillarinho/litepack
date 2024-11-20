package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestVacuumWithTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "Expected no error while creating sqlmock")
	defer db.Close()

	ctx := context.Background()

	t.Run("should execute VACUUM successfully", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("VACUUM;").
			WillReturnResult(sqlmock.NewResult(0, 0))

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		db := &database{}
		err = db.Vacuum(ctx, tx)

		assert.NoError(t, err, "Expected no error while executing VACUUM")
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})

	t.Run("should return an error if VACUUM fails", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("VACUUM;").
			WillReturnError(fmt.Errorf("mock vacuum error"))

		tx, err := db.Begin()
		assert.NoError(t, err, "Expected no error while starting transaction")

		db := &database{}
		err = db.Vacuum(ctx, tx)

		assert.Error(t, err, "Expected an error when VACUUM fails")
		assert.Equal(
			t,
			"vacuuming: mock vacuum error",
			err.Error(),
			"Expected error message to match",
		)
		assert.NoError(t, mock.ExpectationsWereMet(), "Not all expectations were met")
	})
}
