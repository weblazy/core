package breaker

import (
	"errors"
	"sync"
	"time"
)

const (
	defaultMaxRequests                = 3
	defaultConsecutiveFailuresToBreak = 5
	defaultInterval                   = time.Second * 5
	defaultTimeout                    = time.Second * 10
)

// ErrTooManyRequests is returned when the CB state is half open and the requests count is over the cb maxRequests
var ErrTooManyRequests = errors.New("too many requests on half-open state")

type (
	// counts holds the numbers of requests and their successes/failures.
	// netflixBreaker clears the internal counts either
	// on the change of the state or at the closed-state intervals.
	// counts ignores the results of the requests sent before clearing.
	counts struct {
		Requests             uint32
		TotalSuccesses       uint32
		TotalFailures        uint32
		ConsecutiveSuccesses uint32
		ConsecutiveFailures  uint32
	}

	// Settings configures netflixBreaker:
	//
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the netflixBreaker is half-open.
	// If MaxRequests is 0, the netflixBreaker allows only 3 request.
	//
	// Interval is the cyclic period of the closed state
	// for the netflixBreaker to clear the internal counts.
	// If Interval is 0, the interval value of the netflixBreaker is set to 5 seconds.
	//
	// Timeout is the period of the open state,
	// after which the state of the netflixBreaker becomes half-open.
	// If Timeout is 0, the timeout value of the netflixBreaker is set to 10 seconds.
	//
	// ReadyToTrip is called with a copy of counts whenever a request fails in the closed state.
	// If ReadyToTrip returns true, the netflixBreaker will be placed into the open state.
	// If ReadyToTrip is nil, default ReadyToTrip is used.
	// Default ReadyToTrip returns true when the number of consecutive failures is more than 5.
	Settings struct {
		MaxRequests uint32
		Interval    time.Duration
		Timeout     time.Duration
		ReadyToTrip func(counts counts) bool
	}

	// netflixBreaker is a state machine to prevent sending requests that are likely to fail.
	netflixBreaker struct {
		maxRequests uint32
		interval    time.Duration
		timeout     time.Duration
		readyToTrip func(counts counts) bool

		mutex      sync.Mutex
		state      State
		generation uint64
		counts     counts
		expiry     time.Time
	}
)

func (c *counts) onRequest() {
	c.Requests++
}

func (c *counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// newNetflixBreaker returns a new netflixBreaker with default settings.
func newNetflixBreaker() *netflixBreaker {
	cb := new(netflixBreaker)
	withSettings(Settings{})(cb)
	cb.toNewGeneration(time.Now())

	return cb
}

// newNetflixBreakerWithSettings returns a new netflixBreaker configured with the given Settings.
func newNetflixBreakerWithSettings(st Settings) *netflixBreaker {
	cb := new(netflixBreaker)
	withSettings(st)(cb)
	cb.toNewGeneration(time.Now())

	return cb
}

func (cb *netflixBreaker) allow() (Promise, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	return netflixPromise{
		b:          cb,
		generation: generation,
	}, nil
}

func (cb *netflixBreaker) doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		if fallback != nil {
			return fallback(err)
		} else {
			return err
		}
	}

	defer func() {
		if e := recover(); e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = req()
	cb.afterRequest(generation, acceptable(err))
	return err
}

func (cb *netflixBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if state == StateOpen {
		return generation, ErrServiceUnavailable
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.onRequest()
	return generation, nil
}

func (cb *netflixBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *netflixBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *netflixBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *netflixBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}

	return cb.state, cb.generation
}

func (cb *netflixBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	cb.state = state
	cb.toNewGeneration(now)
}

func (cb *netflixBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

type netflixPromise struct {
	b          *netflixBreaker
	generation uint64
}

func (p netflixPromise) Accept() {
	p.b.afterRequest(p.generation, true)
}

func (p netflixPromise) Reject() {
	p.b.afterRequest(p.generation, false)
}

func readyToTrip(counts counts) bool {
	return counts.ConsecutiveFailures > defaultConsecutiveFailuresToBreak
}

func withSettings(st Settings) func(*netflixBreaker) {
	return func(cb *netflixBreaker) {
		if st.MaxRequests == 0 {
			cb.maxRequests = defaultMaxRequests
		} else {
			cb.maxRequests = st.MaxRequests
		}

		if st.Interval == 0 {
			cb.interval = defaultInterval
		} else {
			cb.interval = st.Interval
		}

		if st.Timeout == 0 {
			cb.timeout = defaultTimeout
		} else {
			cb.timeout = st.Timeout
		}

		if st.ReadyToTrip == nil {
			cb.readyToTrip = readyToTrip
		} else {
			cb.readyToTrip = st.ReadyToTrip
		}
	}
}
