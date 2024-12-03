package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/database"
)

func TestDatabase(t *testing.T) {
	t.Skip("Not implemented")

	ctx := context.Background()
	db, err := database.NewDatabase(ctx, "", "test.db")
	assert.Nil(t, err, "Failed to initialize database")
	defer db.Destroy(ctx)

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

}
