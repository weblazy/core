package clientinterceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

const defaultTimeout = time.Second * 2

func ForTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
