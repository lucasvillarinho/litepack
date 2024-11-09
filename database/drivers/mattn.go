package drivers

import (
	"database/sql"
	"fmt"

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
