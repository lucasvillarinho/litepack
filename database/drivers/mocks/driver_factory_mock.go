package drivers

import "github.com/lucasvillarinho/litepack/database/drivers"

type MockDriverFactory struct {
	MockDriver drivers.Driver
	Error      error
}

func (m *MockDriverFactory) GetDriver(
	driverType drivers.DriverType,
	dsn string,
) (drivers.Driver, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.MockDriver, nil
}
