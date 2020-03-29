package threading

import (
	"lazygo/core/fx"
)

type TaskRunner struct {
	limitChan chan struct{}
}

func NewTaskRunner(concurrency int) *TaskRunner {
	return &TaskRunner{
		limitChan: make(chan struct{}, concurrency),
	}
}

func (rp *TaskRunner) Schedule(task func()) {
	rp.limitChan <- struct{}{}

	go func() {
		defer fx.Rescue(func() {
			<-rp.limitChan
		})

		task()
	}()
}
