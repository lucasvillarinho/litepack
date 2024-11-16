package drivers

import "database/sql"

type MockEngine struct {
	QueryErr        error
	queryResultRows *sql.Rows
	queryRowResult  *sql.Row
	ExecutedQuery   string
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

func (m *MockEngine) Execute(query string, args ...interface{}) (sql.Result, error) {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return nil, m.QueryErr
}

func (m *MockEngine) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.queryResultRows, nil
}

func (m *MockEngine) QueryRow(query string, args ...interface{}) *sql.Row {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return m.queryRowResult
}

func (m *MockEngine) Close() error {
	return nil
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
