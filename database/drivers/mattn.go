package drivers

import (
	"database/sql"
	"fmt"

	// Import the sqlite3 driver to register it with the database/sql package.
	_ "github.com/mattn/go-sqlite3"
)

type driverMattn struct {
	BaseDriver
}

func NewMattnDriver(dsn string) (Driver, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &driverMattn{
		BaseDriver: BaseDriver{
			DB: db,
		},
	}, nil
}
