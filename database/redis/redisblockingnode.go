package redis

import (
	"fmt"

	"lazygo/core/logx"

	red "github.com/go-redis/redis"
)

type ClosableNode interface {
	RedisNode
	Close()
}

func CreateBlockingNode(r *Redis) (ClosableNode, error) {
	timeout := readWriteTimeout + blockingQueryTimeout

	switch r.RedisType {
	case NodeType:
		client := red.NewClient(&red.Options{
			Addr:         r.RedisAddr,
			Password:     r.RedisPass,
			DB:           defaultDatabase,
			MaxRetries:   maxRetries,
			PoolSize:     1,
			MinIdleConns: 1,
			ReadTimeout:  timeout,
		})
		return &clientBridge{client}, nil
	case ClusterType:
		client := red.NewClusterClient(&red.ClusterOptions{
			Addrs:        []string{r.RedisAddr},
			Password:     r.RedisPass,
			MaxRetries:   maxRetries,
			PoolSize:     1,
			MinIdleConns: 1,
			ReadTimeout:  timeout,
		})
		return &clusterBridge{client}, nil
	default:
		return nil, fmt.Errorf("unknown redis type: %s", r.RedisType)
	}
}

type (
	clientBridge struct {
		*red.Client
	}

	clusterBridge struct {
		*red.ClusterClient
	}
)

func (bridge *clientBridge) Close() {
	if err := bridge.Client.Close(); err != nil {
		logx.Errorf("Error occurred on close redis client: %s", err)
	}
}

func (bridge *clusterBridge) Close() {
	if err := bridge.ClusterClient.Close(); err != nil {
		logx.Errorf("Error occurred on close redis cluster: %s", err)
	}
}
