package drivers

import "database/sql"

type Driver interface {
	Execute(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
	Close() error
}

// DriverBase is a base implementation that satisfies the Driver interface.
type BaseDriver struct {
	DB *sql.DB
}

// Execute executes a command that does not return rows, such as INSERT, UPDATE, or DELETE.
func (d *BaseDriver) Execute(query string, args ...interface{}) (sql.Result, error) {
	return d.DB.Exec(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (d *BaseDriver) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.Query(query, args...)
}

// QueryRow executes a SELECT command that returns a single row.
func (d *BaseDriver) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRow(query, args...)
}

// Close closes the database connection.
func (d *BaseDriver) Close() error {
	return d.DB.Close()
}

// Begin starts a new transaction.
func (d *BaseDriver) Begin() (*sql.Tx, error) {
	return d.DB.Begin()
}
