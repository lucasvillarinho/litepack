package database

import (
	"fmt"

	"github.com/lucasvillarinho/litepack/database/drivers"
)

type DriverType string

const (
	// DriverMattn "github.com/mattn/go-sqlite3".
	DriverMattn DriverType = "mattn"
	// DriverModernc r "modernc.org/sqlite".
	DriverModernc DriverType = "modernc"
)

var supportedDrivers = map[DriverType]func(string) (drivers.Driver, error){
	DriverMattn:   drivers.NewMattnDriver,
	DriverModernc: drivers.NewModerncDriver,
}

// NewDriverFactory creates a new instance of DriverFactory.
func NewEngine(dt DriverType, dsn string) (drivers.Driver, error) {
	createDriverFunc, exists := supportedDrivers[dt]
	if !exists {
		return nil, fmt.Errorf("unsupported driver type: %s", dt)
	}

	driver, err := createDriverFunc(dsn)
	if err != nil {
		return nil, fmt.Errorf("error creating driver: %w", err)
	}

	return driver, nil
}
