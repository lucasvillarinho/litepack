package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/database/drivers"
)

func TestDatabase(t *testing.T) {

	ctx := context.Background()
	db, err := database.NewDatabase(ctx, "", "test.db")
	assert.Nil(t, err, "Failed to initialize database")

	t.Run("Should execute a simple query", func(t *testing.T) {
		query := `CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, value TEXT)`

		err := db.Exec(ctx, query)

		assert.Nil(t, err, "Expected Exec to run successfully, but got: %v", err)
	})

	t.Run("Should insert and retrieve data", func(t *testing.T) {
		insertQuery := `INSERT INTO test_table (value) VALUES (?)`

		err := db.Exec(ctx, insertQuery, "test_value")
		assert.Nil(t, err, "Expected insert query to succeed, but got: %v", err)

		selectQuery := `SELECT value FROM test_table WHERE id = 1`
		var value string
		err = db.ExecWithTx(ctx, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, selectQuery).Scan(&value)
		})

		assert.Nil(t, err, "Expected select query to succeed, but got: %v", err)
		assert.Equal(t, "test_value", value, "Expected retrieved value to be 'test_value', but got: %v", value)
	})

	t.Run("Close", func(t *testing.T) {
		db, err := database.NewDatabase(ctx, "", "test.db")
		assert.Nil(t, err, "Failed to reinitialize database")

		err = db.Close(ctx)

		assert.Nil(t, err, "Expected Close to succeed, but got: %v", err)
	})

	t.Run("GetEngine", func(t *testing.T) {
		engine := db.GetEngine(ctx)

		assert.NotNil(t, engine, "Expected GetEngine to return a valid driver")
		assert.Implements(t, (*drivers.Driver)(nil), engine, "Expected GetEngine to return an instance of drivers.Driver")
	})

	t.Run("Exec", func(t *testing.T) {
		query := `CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, value TEXT)`
		err := db.Exec(ctx, query)
		assert.Nil(t, err, "Expected Exec to run successfully, but got: %v", err)

		insertQuery := `INSERT INTO test_table (value) VALUES (?)`
		err = db.Exec(ctx, insertQuery, "test_value")

		assert.Nil(t, err, "Expected insert query to succeed, but got: %v", err)
	})

	t.Run("ExecWithTx", func(t *testing.T) {
		selectQuery := `SELECT value FROM test_table WHERE id = ?`
		var value string
		err := db.ExecWithTx(ctx, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, selectQuery, 1).Scan(&value)
		})

		assert.Nil(t, err, "Expected ExecWithTx to succeed, but got: %v", err)
		assert.Equal(t, "test_value", value, "Expected retrieved value to be 'test_value', but got: %v", value)
	})

	t.Run("Vacuum", func(t *testing.T) {
		err := db.ExecWithTx(ctx, func(tx *sql.Tx) error {
			return db.Vacuum(ctx)
		})

		assert.Nil(t, err, "Expected Vacuum to succeed, but got: %v", err)
	})
}
