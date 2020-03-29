package internal

import (
	"encoding/json"
	"errors"

	"lazygo/core/database/redis"
	"lazygo/core/logx"
	"lazygo/core/syncx"
)

const (
	notFoundExpiry      = 60 // seconds
	notFoundPlaceholder = "*"
)

// indicates there is no such value associate with the key
var ErrPlaceholder = errors.New("placeholder")

type Cache struct {
	rds         *redis.Redis
	barrier     syncx.SharedCalls
	stat        *CacheStat
	errNotFound error
}

func NewCache(rds *redis.Redis, barrier syncx.SharedCalls, stat *CacheStat, errNotFound error) Cache {
	return Cache{
		rds:         rds,
		barrier:     barrier,
		stat:        stat,
		errNotFound: errNotFound,
	}
}

func (c Cache) DelCache(key string) error {
	_, err := c.rds.Del(key)
	return err
}

func (c Cache) SetCache(key string, v interface{}, seconds int) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if seconds > 0 {
		return c.rds.Setex(key, string(data), seconds)
	} else {
		return c.rds.Set(key, string(data))
	}
}

func (c Cache) SetCacheWithNotFound(key string) error {
	return c.rds.Setex(key, notFoundPlaceholder, notFoundExpiry)
}

func (c Cache) Take(v interface{}, key string, seconds int, query func(v interface{}) error) error {
	c.stat.IncrementTotal()
	val, fresh, err := c.barrier.DoEx(key, func() (interface{}, error) {
		if err := c.queryCache(key, v); err != nil {
			if err == ErrPlaceholder {
				c.stat.IncrementCache()
				return nil, c.errNotFound
			} else if err != c.errNotFound {
				c.stat.IncrementCacheFails()
				// why we just return the error instead of query from db,
				// because we don't allow the disaster pass to the dbs.
				// fail fast, in case we bring down the dbs.
				return nil, err
			}

			if err = query(v); err == c.errNotFound {
				if err = c.SetCacheWithNotFound(key); err != nil {
					logx.Error(err)
				}

				return nil, c.errNotFound
			} else if err != nil {
				c.stat.IncrementDbFails()
				return nil, err
			}

			if err = c.SetCache(key, v, seconds); err != nil {
				logx.Error(err)
			}
		} else {
			// successfully queried from cache
			c.stat.IncrementCache()
		}

		return json.Marshal(v)
	})
	if err != nil {
		return err
	}
	if fresh {
		return nil
	} else {
		// got the result from previous ongoing query
		c.stat.IncrementCache()
	}

	return json.Unmarshal(val.([]byte), v)
}

func (c Cache) queryCache(key string, v interface{}) error {
	data, err := c.rds.Get(key)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return c.errNotFound
	}

	if data == notFoundPlaceholder {
		return ErrPlaceholder
	}

	return json.Unmarshal([]byte(data), v)
}
