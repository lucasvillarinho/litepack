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
	dsn    string
}

type Database interface {
	Destroy(ctx context.Context) error
	Close(ctx context.Context) error
	Vacuum(ctx context.Context) error
	GetEngine(ctx context.Context) drivers.Driver
	ExecWithTx(ctx context.Context, fn func(*sql.Tx) error) error
	Exec(ctx context.Context, query string, args ...interface{}) error

	SetJournalModeWal(ctx context.Context) error
	SetPageSize(ctx context.Context, pageSize int) error
	SetCacheSize(ctx context.Context, cacheSize int) error
	SetMaxPageCount(ctx context.Context, pageCount int) error
	SetEngine(ctx context.Context, driver Driver) error
}

// NewDatabase creates a new database instance with the given DSN and applies any provided options.
func NewDatabase(ctx context.Context, path, dbName string) (Database, error) {
	db := &database{}

	dsn, err := helpers.CreateDSN(path, dbName)
	if err != nil {
		return nil, fmt.Errorf("error creating DSN: %w", err)
	}
	db.dsn = dsn

	err = db.SetEngine(ctx, DriverMattn)
	if err != nil {
		return nil, fmt.Errorf("error setting up engine: %w", err)
	}

	return db, nil
}

// SetJournalMode sets the journal mode to WAL.
//
// Parameters:
//   - ctx: the context
//
// Returns:
//   - error: an error if the operation failed
func (db *database) SetJournalModeWal(ctx context.Context) error {
	_, err := db.engine.ExecContext(ctx, "PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("enabling WAL mode: %w", err)
	}

	return nil
}

// SetPageSize sets the page size.
//
// Parameters:
//   - ctx: the context
//   - pageSize: the page size
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	db := database.NewDatabase(ctx, "path/to/database", "db.sqlite")
//	defer db.Close(ctx)
//	err := db.SetPageSize(ctx, 4096) // 4096 bytes
//	if err != nil {
//		return err
//	}
func (db *database) SetPageSize(ctx context.Context, pageSize int) error {
	if pageSize == 0 {
		return fmt.Errorf("invalid page size: %d", pageSize)
	}

	_, err := db.engine.ExecContext(ctx, fmt.Sprintf("PRAGMA page_size = %d;", pageSize))
	if err != nil {
		return fmt.Errorf("setting page size: %w", err)
	}

	return nil
}

// Set CacheSize sets the page size.
// Cache size is the number of pages in the cache.
// cacheSize = cacheSize/pageSize
//
// Parameters:
//   - ctx: the context
//   - cacheSize: the cache size
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	db := database.NewDatabase(ctx, "path/to/database", "db.sqlite")
//	defer db.Close(ctx)
//	err := db.SetCacheSize(ctx, 1000)
//	if err != nil {
//		return err
//	}
func (db *database) SetCacheSize(ctx context.Context, cacheSize int) error {
	if cacheSize == 0 {
		return fmt.Errorf("invalid cache size or page size: %d", cacheSize)
	}

	_, err := db.engine.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA cache_size = %d;", cacheSize),
	)
	if err != nil {
		return fmt.Errorf("setting cache size: %w", err)
	}

	return nil
}

// SetMaxPageCount sets the max page count
//
// Parameters:
//   - ctx: the context
//   - maxPageCount: the max page count
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	db := database.NewDatabase(ctx, "path/to/database", "db.sqlite")
//	defer db.Close(ctx)
//	err := db.SetMaxPageCount(ctx, 1000)
//	if err != nil {
//		return err
//	}
func (db *database) SetMaxPageCount(ctx context.Context, maxPageCount int) error {
	if maxPageCount == 0 {
		return fmt.Errorf("invalid max page count: %d", maxPageCount)
	}

	_, err := db.engine.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA max_page_count = %d;", maxPageCount),
	)
	if err != nil {
		return fmt.Errorf("setting max page count: %w", err)
	}

	return nil
}

// SetEngine creates a new database engine with the given driver and DSN.
//
// Parameters:
//   - ctx: the context
//   - driver: the database driver
//
// Returns:
//   - error: an error if the operation failed
//
// Example:
//
//	db := database.NewDatabase(ctx, "path/to/database", "db.sqlite")
//	defer db.Close(ctx)
//
//	err := db.SetEngine(ctx, database.DriverMattn)
//	if err != nil {
//		return err
//	}
func (db *database) SetEngine(ctx context.Context, driver Driver) error {
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
//
// Returns:
//   - error: an error if the operation failed
//
// ⚠️ WARNING: This operation may take a long time to complete on large databases.
func (db *database) Vacuum(ctx context.Context) error {
	_, err := db.engine.ExecContext(ctx, "VACUUM;")
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
func (db *database) ExecWithTx(_ context.Context, fn func(*sql.Tx) error) error {
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
