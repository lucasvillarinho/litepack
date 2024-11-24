package log

import (
	"context"
	"fmt"

	"github.com/lucasvillarinho/litepack/database"
	"github.com/lucasvillarinho/litepack/internal/log/queries"
)

type Level string

const (
	LevelError Level = "ERROR"
)

type Logger interface {
	Error(ctx context.Context, msg string)
}

type logger struct {
	database database.Database
	queries  *queries.Queries
}

// NewLogger creates a new logger instance.
// The logger is backed by a database.
//
// Parameters:
//   - ctx: the context
//   - db: the database
//
// Returns:
//   - logger: the logger instance
//   - error: an error if the operation failed
//
// Warning: only error messages are supported.
//
// Example:
//
//	db, err := database.NewDatabase("sqlite3", "file.db")
//	if err != nil {
//	  return err
//	}
//	logger, err := log.NewLogger(db)
//	if err != nil {
//	  return err
//	}
//	logger.Error(ctx, "an error occurred")
func NewLogger(ctx context.Context, db database.Database) (Logger, error) {
	lg := &logger{
		database: db,
	}

	lg.queries = queries.New(db.GetEngine(ctx))

	err := lg.queries.CreateLogTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create log table: %w", err)
	}

	return lg, nil
}

// Error logs an error message.
//
// Parameters:
//   - ctx: the context
//   - msg: the error message
//
// Example:
//
//	logger.Error(ctx, "an error occurred")
func (lg *logger) Error(ctx context.Context, msg string) {
	paransInsert := queries.InsertLogParams{
		Level:   string(LevelError),
		Message: msg,
	}

	_ = lg.queries.InsertLog(ctx, paransInsert)
}
