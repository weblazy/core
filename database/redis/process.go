package redis

import (
	"strings"

	"lazygo/core/logx"
	"lazygo/core/mapping"
	"time"

	red "github.com/go-redis/redis"
)

func process(proc func(red.Cmder) error) func(red.Cmder) error {
	return func(cmd red.Cmder) error {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			if duration > slowThreshold {
				var buf strings.Builder
				buf.WriteString(cmd.Name())
				for _, arg := range cmd.Args() {
					buf.WriteString(mapping.Repr(arg))
				}
				logx.WithDuration(duration).Slowf("[REDIS] slowcall on executing: %s", buf.String())
			}
		}()

		return proc(cmd)
	}
}
