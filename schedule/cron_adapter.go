package schedule

import "github.com/robfig/cron/v3"

type cronAdapter struct {
	cron *cron.Cron
}

func (c *cronAdapter) AddFunc(spec string, cmd func()) (int, error) {
	id, err := c.cron.AddFunc(spec, cmd)
	return int(id), err
}

func (c *cronAdapter) Start() {
	c.cron.Start()
}

func (c *cronAdapter) Stop() {
	c.cron.Stop()
}

func (c *cronAdapter) Entries() []cron.Entry {
	return c.cron.Entries()
}
