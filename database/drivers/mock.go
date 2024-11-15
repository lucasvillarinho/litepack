package drivers

import "database/sql"

type Mock struct {
	QueryErr        error
	queryResultRows *sql.Rows
	queryRowResult  *sql.Row
	ExecutedQuery   string
	ExecutedArgs    []interface{}
}

func (m *Mock) Execute(query string, args ...interface{}) (sql.Result, error) {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return nil, m.QueryErr
}

func (m *Mock) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.queryResultRows, nil
}

func (m *Mock) QueryRow(query string, args ...interface{}) *sql.Row {
	m.ExecutedQuery = query
	m.ExecutedArgs = args
	return m.queryRowResult
}

func (m *Mock) Close() error {
	return nil
}
