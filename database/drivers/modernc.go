package drivers

import (
	"database/sql"
	"fmt"

	// Import the sqlite driver to register it with the database/sql package.
	_ "modernc.org/sqlite"
)

type driverModernc struct {
	BaseDriver
}

func NewModerncDriver(dsn string) (Driver, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &driverModernc{
		BaseDriver: BaseDriver{
			DB: db,
		},
	}, nil
}
