package drivers

import "database/sql"

type MockEngine struct {
	QueryErr        error
	queryResultRows *sql.Rows
	queryRowResult  *sql.Row
	ExecutedQuery   string
	ExecutedArgs    []interface{}
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
