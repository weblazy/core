package breaker

import (
	"errors"
	"fmt"
	"os"

	"lazygo/core/stringx"
)

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// ErrServiceUnavailable is returned when the CB state is open
var ErrServiceUnavailable = errors.New("circuit breaker is open")

type (
	State      = int32
	Acceptable func(err error) bool

	Breaker interface {
		// Name returns the name of the netflixBreaker.
		Name() string

		// Allow checks if the request is allowed.
		// If allowed, a promise will be returned, the caller needs to call promise.Accept()
		// on success, or call promise.Reject() on failure.
		// If not allow, ErrServiceUnavailable will be returned.
		Allow() (Promise, error)

		// Do runs the given request if the netflixBreaker accepts it.
		// Do returns an error instantly if the netflixBreaker rejects the request.
		// If a panic occurs in the request, the netflixBreaker handles it as an error
		// and causes the same panic again.
		Do(req func() error) error

		// DoWithAcceptable runs the given request if the netflixBreaker accepts it.
		// Do returns an error instantly if the netflixBreaker rejects the request.
		// If a panic occurs in the request, the netflixBreaker handles it as an error
		// and causes the same panic again.
		// acceptable checks if it's a successful call, even if the err is not nil.
		DoWithAcceptable(req func() error, acceptable Acceptable) error

		// DoWithFallback runs the given request if the netflixBreaker accepts it.
		// DoWithFallback runs the fallback if the netflixBreaker rejects the request.
		// If a panic occurs in the request, the netflixBreaker handles it as an error
		// and causes the same panic again.
		DoWithFallback(req func() error, fallback func(err error) error) error

		// DoWithFallbackAcceptable runs the given request if the netflixBreaker accepts it.
		// DoWithFallback runs the fallback if the netflixBreaker rejects the request.
		// If a panic occurs in the request, the netflixBreaker handles it as an error
		// and causes the same panic again.
		// acceptable checks if it's a successful call, even if the err is not nil.
		DoWithFallbackAcceptable(req func() error, fallback func(err error) error, acceptable Acceptable) error
	}

	BreakerOption func(breaker *circuitBreaker)

	Promise interface {
		Accept()
		Reject()
	}

	circuitBreaker struct {
		name string
		throttle
	}

	throttle interface {
		allow() (Promise, error)
		doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error
	}
)

func NewBreaker(opts ...BreakerOption) Breaker {
	var b circuitBreaker
	for _, opt := range opts {
		opt(&b)
	}
	if len(b.name) == 0 {
		b.name = stringx.Rand()
	}
	if b.throttle == nil {
		b.throttle = newGoogleBreaker()
	}
	b.throttle = loggedThrottle{
		name:     b.name,
		throttle: b.throttle,
	}

	return &b
}

func (cb *circuitBreaker) Allow() (Promise, error) {
	return cb.throttle.allow()
}

func (cb *circuitBreaker) Do(req func() error) error {
	return cb.throttle.doReq(req, nil, defaultAcceptable)
}

func (cb *circuitBreaker) DoWithAcceptable(req func() error, acceptable Acceptable) error {
	return cb.throttle.doReq(req, nil, acceptable)
}

func (cb *circuitBreaker) DoWithFallback(req func() error, fallback func(err error) error) error {
	return cb.throttle.doReq(req, fallback, defaultAcceptable)
}

func (cb *circuitBreaker) DoWithFallbackAcceptable(req func() error, fallback func(err error) error,
	acceptable Acceptable) error {
	return cb.throttle.doReq(req, fallback, acceptable)
}

func (cb *circuitBreaker) Name() string {
	return cb.name
}

func GoogleAlgorithm() BreakerOption {
	return func(b *circuitBreaker) {
		b.throttle = newGoogleBreaker()
	}
}

func NetflixAlgorithm() BreakerOption {
	return func(b *circuitBreaker) {
		b.throttle = newNetflixBreaker()
	}
}

func WithName(name string) BreakerOption {
	return func(b *circuitBreaker) {
		b.name = name
	}
}

func defaultAcceptable(err error) bool {
	return err == nil
}

type loggedThrottle struct {
	name string
	throttle
}

func (lt loggedThrottle) allow() (Promise, error) {
	promise, err := lt.throttle.allow()
	return promise, lt.logError(err)
}

func (lt loggedThrottle) doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error {
	return lt.logError(lt.throttle.doReq(req, fallback, acceptable))
}

func (lt loggedThrottle) logError(err error) error {
	if err == ErrServiceUnavailable {
		if hostname, err := os.Hostname(); err == nil {
			fmt.Println(fmt.Sprintf("host: %s, callee: %s, circuit breaker is open and requests dropped",
				hostname, lt.name))
		}
	}

	return err
}
