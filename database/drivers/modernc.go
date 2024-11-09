package drivers

import (
	"database/sql"
	"fmt"

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
