package rpcx

import (
	"time"

	"github.com/weblazy/core/rpcx/clientinterceptors"

	"google.golang.org/grpc"
)

type (
	ClientOptions struct {
		Timeout     time.Duration
		DialOptions []grpc.DialOption
	}

	ClientOption func(options *ClientOptions)

	Client interface {
		Next() (*grpc.ClientConn, bool)
	}
)

func WithDialOption(opt grpc.DialOption) ClientOption {
	return func(options *ClientOptions) {
		options.DialOptions = append(options.DialOptions, opt)
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(options *ClientOptions) {
		options.Timeout = timeout
	}
}

func buildDialOptions(opts ...ClientOption) []grpc.DialOption {
	var clientOptions ClientOptions
	for _, opt := range opts {
		opt(&clientOptions)
	}

	options := []grpc.DialOption{
		grpc.WithInsecure(),
		WithUnaryClientInterceptors(
			clientinterceptors.BreakerInterceptor,
			clientinterceptors.DurationInterceptor,
			clientinterceptors.ForTimeoutInterceptor(clientOptions.Timeout),
		),
	}

	return append(options, clientOptions.DialOptions...)
}
