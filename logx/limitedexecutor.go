package logx

import (
	"sync/atomic"
	"time"
)

type limitedExecutor struct {
	threshold int64
	lastTime  *int64
	discarded uint32
}

func newLimitedExecutor(milliseconds int) *limitedExecutor {
	return &limitedExecutor{
		threshold: int64(milliseconds) * 1000000,
	}
}

func (le *limitedExecutor) logOrDiscard(execute func()) {
	if le == nil || le.threshold <= 0 {
		execute()
		return
	}

	now := time.Now().UnixNano()
	if now-atomic.LoadInt64(le.lastTime) <= le.threshold {
		atomic.AddUint32(&le.discarded, 1)
	} else {
		atomic.StoreInt64(le.lastTime, now)
		discarded := atomic.SwapUint32(&le.discarded, 0)
		if discarded > 0 {
			Errorf("Discarded %d error messages", discarded)
		}

		execute()
	}
}
