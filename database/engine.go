package database

import (
	"fmt"

	"github.com/lucasvillarinho/litepack/database/drivers"
)

type Driver string

const (
	// DriverMattn "github.com/mattn/go-sqlite3".
	DriverMattn Driver = "mattn"
	// DriverModernc r "modernc.org/sqlite".
	DriverModernc Driver = "modernc"
)

var supportedDrivers = map[Driver]func(string) (drivers.Driver, error){
	DriverMattn:   drivers.NewMattnDriver,
	DriverModernc: drivers.NewModerncDriver,
}

// NewEngine creates a new instance of DriverFactory.
func NewEngine(dt Driver, dsn string) (drivers.Driver, error) {
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
