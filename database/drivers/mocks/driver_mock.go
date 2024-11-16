package drivers

import (
	"context"
	"database/sql"
)

type MockEngine struct {
	QueryErr        error
	PrepareErr      error
	BeginError      error
	QueryErrors     map[string]error
	QueryResultRows *sql.Rows
	QueryRowResult  *sql.Row
	TxMock          *MockTx
	PrepareQuery    string
	ExecutedQuery   string
	ExecutedQueries []string
	ExecutedArgs    []interface{}
	BeginCalled     bool
}

type MockTx struct {
	ExecQueries []string
	ExecArgs    [][]interface{}
	Committed   bool
	RolledBack  bool
}

func (m *MockEngine) ExecContext(
	_ context.Context,
	query string,
	args ...interface{},
) (sql.Result, error) {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args

	if err, ok := m.QueryErrors[query]; ok {
		return nil, err
	}

	return nil, m.QueryErr
}

func (m *MockEngine) QueryContext(
	_ context.Context,
	query string,
	args ...interface{},
) (*sql.Rows, error) {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.QueryResultRows, nil
}

func (m *MockEngine) QueryRowContext(
	_ context.Context,
	query string,
	args ...interface{},
) *sql.Row {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return m.QueryRowResult
}

func (m *MockEngine) PrepareContext(_ context.Context, query string) (*sql.Stmt, error) {
	m.PrepareQuery = query
	return nil, m.PrepareErr
}

func (m *MockEngine) Close() error {
	return nil
}

func (m *MockEngine) Begin() (*sql.Tx, error) {
	m.BeginCalled = true
	if m.BeginError != nil {
		return nil, m.BeginError
	}
	if m.TxMock == nil {
		m.TxMock = &MockTx{}
	}
	return &sql.Tx{}, nil
}

func (m *MockTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.ExecQueries = append(m.ExecQueries, query)
	m.ExecArgs = append(m.ExecArgs, args)
	return nil, nil
}

func (m *MockTx) Commit() error {
	m.Committed = true
	return nil
}

func (m *MockTx) Rollback() error {
	m.RolledBack = true
	return nil
}
