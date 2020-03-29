package redis

import (
	"io"

	"lazygo/core/syncx"

	red "github.com/go-redis/redis"
)

const (
	defaultDatabase = 0
	maxRetries      = 3
	idleConns       = 8
)

var clientManager = syncx.NewResourceManager()

func getClient(server, pass string) (*red.Client, error) {
	val, err := clientManager.GetResource(server, func() (io.Closer, error) {
		store := red.NewClient(&red.Options{
			Addr:         server,
			Password:     pass,
			DB:           defaultDatabase,
			MaxRetries:   maxRetries,
			MinIdleConns: idleConns,
		})
		store.WrapProcess(process)
		return store, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(*red.Client), nil
}
