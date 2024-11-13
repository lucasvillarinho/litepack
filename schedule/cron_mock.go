package schedule

import "github.com/robfig/cron/v3"

type mockCron struct {
	addFuncErr   error
	addFuncCalls []struct {
		cmd  func()
		spec string
	}
}

func (m *mockCron) AddFunc(spec string, cmd func()) (int, error) {
	if m.addFuncErr != nil {
		return 0, m.addFuncErr
	}

	m.addFuncCalls = append(m.addFuncCalls, struct {
		cmd  func()
		spec string
	}{cmd, spec})
	return 1, m.addFuncErr
}

func (m *mockCron) Start()                {}
func (m *mockCron) Stop()                 {}
func (m *mockCron) Entries() []cron.Entry { return nil }
