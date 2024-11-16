package drivers

import (
	"context"
	"database/sql"
)

type MockEngine struct {
	QueryErr        error
	QueryErrors     map[string]error
	QueryResultRows *sql.Rows
	QueryRowResult  *sql.Row
	PrepareQuery    string
	PrepareErr      error
	ExecutedQuery   string
	ExecutedQueries []string
	ExecutedArgs    []interface{}
	BeginCalled     bool
	BeginError      error
	TxMock          *MockTx
}

type MockTx struct {
	Committed   bool
	RolledBack  bool
	ExecQueries []string
	ExecArgs    [][]interface{}
}

func (m *MockEngine) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args

	if err, ok := m.QueryErrors[query]; ok {
		return nil, err
	}

	return nil, m.QueryErr
}

func (m *MockEngine) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.QueryResultRows, nil
}

func (m *MockEngine) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return m.QueryRowResult
}

// Simula o método PrepareContext
func (m *MockEngine) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	m.PrepareQuery = query
	return nil, m.PrepareErr
}

// Simula o fechamento do banco
func (m *MockEngine) Close() error {
	return nil
}

// Simula o início de uma transação
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

// Métodos do MockTx
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
