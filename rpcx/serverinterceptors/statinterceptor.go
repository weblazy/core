package serverinterceptors

import (
	"context"
	"encoding/json"
	"time"

	"lazygo/core/logx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const serverSlowThreshold = time.Millisecond * 500

func UnaryStatInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer handleCrash(func(r interface{}) {
			err = toPanicError(r)
		})

		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime)
			logDuration(ctx, info.FullMethod, req, duration)
		}()

		return handler(ctx, req)
	}
}

func logDuration(ctx context.Context, method string, req interface{}, duration time.Duration) {
	var addr string
	client, ok := peer.FromContext(ctx)
	if ok {
		addr = client.Addr.String()
	}
	content, err := json.Marshal(req)
	if err != nil {
		logx.Errorf("%s - %s", addr, err.Error())
	} else if duration > serverSlowThreshold {
		logx.WithDuration(duration).Slowf("[RPC] slowcall - %s - %s - %s", addr, method, string(content))
	} else {
		logx.WithDuration(duration).Infof("%s - %s - %s", addr, method, string(content))
	}
}
