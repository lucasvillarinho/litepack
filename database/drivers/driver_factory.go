package drivers

import "fmt"

type DriverType string

const (
	// DriverMattn "github.com/mattn/go-sqlite3".
	DriverMattn DriverType = "mattn"

	// DriverModernc r "modernc.org/sqlite".
	DriverModernc DriverType = "modernc"
)

// DriverFactory is a factory for creating new instances of drivers.
type DriverFactory struct {
	drivers map[DriverType]func(string) (Driver, error)
}

// NewDriverFactory creates a new instance of DriverFactory.
func NewDriverFactory() *DriverFactory {
	return &DriverFactory{
		drivers: map[DriverType]func(string) (Driver, error){
			DriverMattn:   NewMattnDriver,
			DriverModernc: NewModerncDriver,
		},
	}
}

// GetDriver returns a new instance of the specified driver.
// The driver type must be one of the supported driver types.
//
// Parameters:
// - driverType: The type of the driver to create.
// - dsn: The data source name.
//
// Supported driver types:
// - DriverMattn: "github.com/mattn/go-sqlite3".
// - DriverModernc: "modernc.org/sqlite".
//
// Returns:
// - Driver: The new driver instance.
// - error: An error if the driver type is not supported.
func (f *DriverFactory) GetDriver(driverType DriverType, dsn string) (Driver, error) {
	constructor, exists := f.drivers[driverType]
	if !exists {
		return nil, fmt.Errorf("unknown driver type: %s", driverType)
	}
	return constructor(dsn)
}
