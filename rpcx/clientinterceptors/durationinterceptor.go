package clientinterceptors

import (
	"context"
	"path"
	"time"

	"lazygo/core/logx"

	"google.golang.org/grpc"
)

const slowThreshold = time.Millisecond * 500

func DurationInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	serverName := path.Join(cc.Target(), method)
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		logx.WithDuration(time.Since(start)).Infof("fail - %s - %v - %s", serverName, req, err.Error())
	} else {
		elapsed := time.Since(start)
		if elapsed > slowThreshold {
			logx.WithDuration(elapsed).Slowf("[RPC] ok - slowcall - %s - %v - %v", serverName, req, reply)
		}
	}

	return err
}
