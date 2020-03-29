package syncx

import (
	"sync"
)

type DoneChan struct {
	done chan struct{}
	once sync.Once
}

func NewDoneChan() *DoneChan {
	return &DoneChan{
		done: make(chan struct{}),
	}
}

func (dc *DoneChan) Close() {
	dc.once.Do(func() {
		close(dc.done)
	})
}

func (dc *DoneChan) Done() chan struct{} {
	return dc.done
}
