package testutils

import (
	"sync"
	"time"
)

type TestClock struct {
	time time.Time
	step time.Duration
	lock sync.Mutex
}

func NewTestClock(t time.Time, step time.Duration) *TestClock {
	return &TestClock{
		time: t,
		step: step,
	}
}
func (c *TestClock) Now() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	res := c.time
	c.time = c.time.Add(c.step)
	return res
}
