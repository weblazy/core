package internal

import (
	"sync/atomic"
	"time"

	"lazygo/core/logx"
)

const statInterval = time.Minute

type CacheStat struct {
	name string
	// export the fields to let the unit tests working,
	// reside in internal package, doesn't matter.
	TotalQueries uint64
	CacheQueries uint64
	CacheFails   uint64
	DbFails      uint64
}

func NewCacheStat(name string) CacheStat {
	ret := CacheStat{
		name: name,
	}
	go ret.statLoop()

	return ret
}

func (cs *CacheStat) IncrementTotal() {
	atomic.AddUint64(&cs.TotalQueries, 1)
}

func (cs *CacheStat) IncrementCache() {
	atomic.AddUint64(&cs.CacheQueries, 1)
}

func (cs *CacheStat) IncrementCacheFails() {
	atomic.AddUint64(&cs.CacheFails, 1)
}

func (cs *CacheStat) IncrementDbFails() {
	atomic.AddUint64(&cs.DbFails, 1)
}

func (cs *CacheStat) statLoop() {
	ticker := time.NewTicker(statInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			total := atomic.SwapUint64(&cs.TotalQueries, 0)
			if total == 0 {
				continue
			}

			cache := atomic.SwapUint64(&cs.CacheQueries, 0)
			percent := 100 * float32(cache) / float32(total)
			cachef := atomic.SwapUint64(&cs.CacheFails, 0)
			dbf := atomic.SwapUint64(&cs.DbFails, 0)
			logx.Statf("(%s) - qpm: %d, cached: %d, cache_percent: %.1f%%, cache_fails: %d, db_fails: %d",
				cs.name, total, cache, percent, cachef, dbf)
		}
	}
}
