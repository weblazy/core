package syncx

import (
	"time"
)

type Cond struct {
	signal chan struct{}
}

func NewCond() *Cond {
	return &Cond{
		signal: make(chan struct{}),
	}
}

// WaitWithTimeout wait for signal return remain wait time or timed out
func (cond *Cond) WaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	begin := time.Now()
	select {
	case <-cond.signal:
		elapsed := time.Since(begin)
		remainTimeout := timeout - elapsed
		return remainTimeout, true
	case <-timer.C:
		return 0, false
	}
}

// Wait for signal
func (cond *Cond) Wait() {
	<-cond.signal
}

// Signal wakes one goroutine waiting on c, if there is any.
func (cond *Cond) Signal() {
	select {
	case cond.signal <- struct{}{}:
	default:
	}
}
