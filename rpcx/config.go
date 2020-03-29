package rpcx

import (
	"lazygo/core/database/redis"
)

type (
	RpcServerConf struct {
		ListenOn      string
		Auth          bool               `json:",default=true"`
		Redis         redis.RedisKeyConf `json:",optional"`
		StrictControl bool               `json:",optional"`
		// pending forever is not allowed
		// never set it to 0, if zero, the underlying will set to 2s automatically
		Timeout int64 `json:",default=2000"`
	}

	RpcClientConf struct {
		Server    string `json:",optional"`
		BlockDial bool   `json:",default=false"`
		App       string
		Token     string
		Timeout   int64 `json:",optional"`
	}
)

func NewDirectClientConf(server, app, token string) RpcClientConf {
	return RpcClientConf{
		Server: server,
		App:    app,
		Token:  token,
	}
}

func (sc RpcServerConf) Validate() error {
	if sc.Auth {
		if err := sc.Redis.Validate(); err != nil {
			return err
		}
	}

	return nil
}
