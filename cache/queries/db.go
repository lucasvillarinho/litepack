// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package queries

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.countCacheEntriesStmt, err = db.PrepareContext(ctx, countCacheEntries); err != nil {
		return nil, fmt.Errorf("error preparing query CountCacheEntries: %w", err)
	}
	if q.createCacheDatabaseStmt, err = db.PrepareContext(ctx, createCacheDatabase); err != nil {
		return nil, fmt.Errorf("error preparing query CreateCacheDatabase: %w", err)
	}
	if q.deleteExpiredCacheStmt, err = db.PrepareContext(ctx, deleteExpiredCache); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteExpiredCache: %w", err)
	}
	if q.deleteKeyStmt, err = db.PrepareContext(ctx, deleteKey); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteKey: %w", err)
	}
	if q.deleteKeysByLimitStmt, err = db.PrepareContext(ctx, deleteKeysByLimit); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteKeysByLimit: %w", err)
	}
	if q.getValueStmt, err = db.PrepareContext(ctx, getValue); err != nil {
		return nil, fmt.Errorf("error preparing query GetValue: %w", err)
	}
	if q.selectKeysToDeleteStmt, err = db.PrepareContext(ctx, selectKeysToDelete); err != nil {
		return nil, fmt.Errorf("error preparing query SelectKeysToDelete: %w", err)
	}
	if q.updateLastAccessedAtStmt, err = db.PrepareContext(ctx, updateLastAccessedAt); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateLastAccessedAt: %w", err)
	}
	if q.upsertCacheStmt, err = db.PrepareContext(ctx, upsertCache); err != nil {
		return nil, fmt.Errorf("error preparing query UpsertCache: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.countCacheEntriesStmt != nil {
		if cerr := q.countCacheEntriesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countCacheEntriesStmt: %w", cerr)
		}
	}
	if q.createCacheDatabaseStmt != nil {
		if cerr := q.createCacheDatabaseStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createCacheDatabaseStmt: %w", cerr)
		}
	}
	if q.deleteExpiredCacheStmt != nil {
		if cerr := q.deleteExpiredCacheStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteExpiredCacheStmt: %w", cerr)
		}
	}
	if q.deleteKeyStmt != nil {
		if cerr := q.deleteKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteKeyStmt: %w", cerr)
		}
	}
	if q.deleteKeysByLimitStmt != nil {
		if cerr := q.deleteKeysByLimitStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteKeysByLimitStmt: %w", cerr)
		}
	}
	if q.getValueStmt != nil {
		if cerr := q.getValueStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getValueStmt: %w", cerr)
		}
	}
	if q.selectKeysToDeleteStmt != nil {
		if cerr := q.selectKeysToDeleteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing selectKeysToDeleteStmt: %w", cerr)
		}
	}
	if q.updateLastAccessedAtStmt != nil {
		if cerr := q.updateLastAccessedAtStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateLastAccessedAtStmt: %w", cerr)
		}
	}
	if q.upsertCacheStmt != nil {
		if cerr := q.upsertCacheStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing upsertCacheStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                       DBTX
	tx                       *sql.Tx
	countCacheEntriesStmt    *sql.Stmt
	createCacheDatabaseStmt  *sql.Stmt
	deleteExpiredCacheStmt   *sql.Stmt
	deleteKeyStmt            *sql.Stmt
	deleteKeysByLimitStmt    *sql.Stmt
	getValueStmt             *sql.Stmt
	selectKeysToDeleteStmt   *sql.Stmt
	updateLastAccessedAtStmt *sql.Stmt
	upsertCacheStmt          *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                       tx,
		tx:                       tx,
		countCacheEntriesStmt:    q.countCacheEntriesStmt,
		createCacheDatabaseStmt:  q.createCacheDatabaseStmt,
		deleteExpiredCacheStmt:   q.deleteExpiredCacheStmt,
		deleteKeyStmt:            q.deleteKeyStmt,
		deleteKeysByLimitStmt:    q.deleteKeysByLimitStmt,
		getValueStmt:             q.getValueStmt,
		selectKeysToDeleteStmt:   q.selectKeysToDeleteStmt,
		updateLastAccessedAtStmt: q.updateLastAccessedAtStmt,
		upsertCacheStmt:          q.upsertCacheStmt,
	}
}
