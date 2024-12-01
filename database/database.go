package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lucasvillarinho/litepack/database/drivers"
	"github.com/lucasvillarinho/litepack/internal/helpers"
)

type database struct {
	engine drivers.Driver
	cfg    *config
	dsn    string
}

type config struct {
	dbSize    int
	cacheSize int
	pageSize  int
}

type Option func(*database, *config)

type Database interface {
	Destroy(ctx context.Context) error
	Close(ctx context.Context) error
	Vacuum(ctx context.Context, tx *sql.Tx) error
	GetEngine(ctx context.Context) drivers.Driver
	ExecWithTx(ctx context.Context, fn func(*sql.Tx) error) error
	Exec(ctx context.Context, query string, args ...interface{}) error
}

// NewDatabase creates a new database instance with the given DSN and applies any provided options.
func NewDatabase(ctx context.Context, path, dbName string, options ...Option) (*database, error) {
	cfg := &config{}
	db := &database{
		cfg: cfg,
	}

	dsn, err := helpers.CreateDSN(path, dbName)
	if err != nil {
		return nil, fmt.Errorf("error creating DSN: %w", err)
	}
	db.dsn = dsn

	err = db.setEngine()
	if err != nil {
		return nil, fmt.Errorf("error setting up engine: %w", err)
	}

	err = db.setupDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up database: %w", err)
	}

	for _, option := range options {
		option(db, cfg)
	}

	return db, nil
}

// setupDatabase sets up the database with the given configuration.
func (db *database) setupDatabase(ctx context.Context) error {
	// Set journal mode to WAL
	_, err := db.engine.ExecContext(ctx, "PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Set synchronous mode to NORMAL
	_, err = db.engine.ExecContext(ctx, "PRAGMA synchronous = NORMAL;")
	if err != nil {
		return fmt.Errorf("setting synchronous mode: %w", err)
	}

	err = db.setPageSize(ctx)
	if err != nil {
		return fmt.Errorf("setting page size: %w", err)
	}

	err = db.setCacheSize(ctx)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	err = db.setPageCount(ctx)
	if err != nil {
		return fmt.Errorf("setting page count: %w", err)
	}

	return nil
}

// SetPageSize sets the page size.
func (db *database) setPageSize(ctx context.Context) error {
	if db.cfg.pageSize == 0 {
		return nil
	}

	_, err := db.engine.ExecContext(ctx, fmt.Sprintf("PRAGMA page_size = %d;", db.cfg.pageSize))
	if err != nil {
		return fmt.Errorf("setting page size: %w", err)
	}

	return nil
}

// SetPageSize sets the page size.
func (db *database) setCacheSize(ctx context.Context) error {
	if db.cfg.cacheSize == 0 {
		return nil
	}

	_, err := db.engine.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA cache_size = %d;", db.cfg.cacheSize/db.cfg.pageSize),
	)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	return nil
}

// SetPageSize sets the page count
func (db *database) setPageCount(ctx context.Context) error {
	if db.cfg.pageSize == 0 || db.cfg.dbSize == 0 {
		return nil
	}

	_, err := db.engine.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA max_page_count = %d;", db.cfg.dbSize/db.cfg.pageSize),
	)
	if err != nil {
		return fmt.Errorf("setting max page count: %w", err)
	}

	return nil
}

// setEngine creates a new database engine with the given driver and DSN.
func (db *database) setEngine() error {
	engine, err := NewEngine(DriverMattn, db.dsn)
	if err != nil {
		return fmt.Errorf("error creating driver: %w", err)
	}
	db.engine = engine

	return nil
}

// Destroy deletes the cache database file and closes the database connection.
//
// parameters:
//   - ctx: the context
//
// ⚠️ WARNING: This operation is irreversible and will delete all data stored in the database.
func (db *database) Destroy(ctx context.Context) error {
	err := db.Close(ctx)
	if err != nil {
		return fmt.Errorf("error closing database: %w", err)
	}

	if err := os.Remove(db.dsn); err != nil {
		return fmt.Errorf("error removing database file: %w", err)
	}

	return nil
}

func (db *database) Close(_ context.Context) error {
	return db.engine.Close()
}

// VacuumWithTransaction runs a VACUUM operation on the database.
// This operation rebuilds the database file, repacking it into a minimal amount of disk space.
// It is recommended to run this operation periodically to keep the database file size small.
//
// Parameters:
//   - ctx: the context
//   - tx: the transaction
//
// Returns:
//   - error: an error if the operation failed
//
// ⚠️ WARNING: This operation may take a long time to complete on large databases.
func (db *database) Vacuum(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec("VACUUM;")
	if err != nil {
		return fmt.Errorf("vacuuming: %w", err)
	}
	return nil
}

// GetEngine returns the database engine.
func (db *database) GetEngine(_ context.Context) drivers.Driver {
	return db.engine
}

// ExecWithTx executes a function with a transaction.
//
// Parameters:
//   - ctx: the context
//   - fn: the function to execute
//
// Returns:
//   - error: an error if the operation failed
func (db *database) ExecWithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.engine.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	err = fn(tx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error rolling back transaction: %w", err)
	}

	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		return errors.Join(err, rollbackErr)
	}

	return nil
}

func IsDBFullError(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "database or disk is full") {
		return true
	}

	return false
}

// Exec executes a query with the given arguments.
//
// Parameters:
//   - ctx: the context
//   - query: the query to execute
//   - args: the query arguments
//
// Returns:
//   - error: an error if the operation failed
func (db *database) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := db.engine.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
