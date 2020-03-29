package rpcx

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/connectivity"
	"lazygo/core/rpcx/auth"
	"time"
)

type DirectClient struct {
	conn *grpc.ClientConn
}

func NewDirectClient(c RpcClientConf, opts ...ClientOption) (*DirectClient, error) {
	options := []ClientOption{
		WithDialOption(grpc.WithPerRPCCredentials(&auth.Credential{
			App:   c.App,
			Token: c.Token,
		})),
	}

	if c.BlockDial {
		options = append(options, WithDialOption(grpc.WithBlock()))
	}
	if c.Timeout > 0 {
		options = append(options, WithTimeout(time.Duration(c.Timeout)*time.Millisecond))
	}
	options = append(options, opts...)

	options = append(options, WithDialOption(grpc.WithBalancerName(roundrobin.Name)))
	ops := buildDialOptions(options...)
	conn, err := grpc.Dial(c.Server, ops...)
	if err != nil {
		return nil, err
	}

	return &DirectClient{
		conn: conn,
	}, nil
}

func (c *DirectClient) Next() (*grpc.ClientConn, bool) {
	state := c.conn.GetState()
	if state == connectivity.Ready {
		return c.conn, true
	} else {
		return nil, false
	}
}
