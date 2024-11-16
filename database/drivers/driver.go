package drivers

import (
	"context"
	"database/sql"
)

type Driver interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) // Adicionado
	Begin() (*sql.Tx, error)
	Close() error
}

type BaseDriver struct {
	DB *sql.DB
}

func (d *BaseDriver) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
}

func (d *BaseDriver) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

func (d *BaseDriver) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (d *BaseDriver) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return d.DB.PrepareContext(ctx, query)
}

func (d *BaseDriver) Begin() (*sql.Tx, error) {
	return d.DB.Begin()
}

func (d *BaseDriver) Close() error {
	return d.DB.Close()
}
