package drivers

import (
	"database/sql"
)

type SQLDBAdapter struct {
	DB *sql.DB
}

func (a *SQLDBAdapter) Execute(query string, args ...interface{}) (sql.Result, error) {
	return a.DB.Exec(query, args...)
}

func (a *SQLDBAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return a.DB.Query(query, args...)
}

func (a *SQLDBAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
	return a.DB.QueryRow(query, args...)
}

func (a *SQLDBAdapter) Close() error {
	return a.DB.Close()
}

func (a *SQLDBAdapter) Begin() (*sql.Tx, error) {
	return a.DB.Begin()
}
