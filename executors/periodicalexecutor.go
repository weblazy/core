package executors

import (
	"reflect"
	"sync"
	"time"

	"lazygo/core/system"
	"lazygo/core/threading"
)

type (
	// A type that satisfies executors.TaskContainer can be used as the underlying
	// container that used to do periodical executions.
	TaskContainer interface {
		// AddTask adds the task into the container.
		// Returns true if the container needs to be flushed after the addition.
		AddTask(task interface{}) bool
		// Execute handles the collected tasks by the container when flushing.
		Execute(tasks interface{})
		// RemoveAll removes the contained tasks, and return them.
		RemoveAll() interface{}
	}

	PeriodicalExecutor struct {
		commander chan interface{}
		interval  time.Duration
		container TaskContainer
		lock      sync.Mutex
	}
)

func NewPeriodicalExecutor(interval time.Duration, container TaskContainer) *PeriodicalExecutor {
	executor := &PeriodicalExecutor{
		// buffer 1 to let the caller go quickly
		commander: make(chan interface{}, 1),
		interval:  interval,
		container: container,
	}
	executor.backgroundFlush()
	system.AddShutdownListener(func() {
		executor.ForceFlush()
	})

	return executor
}

func (pe *PeriodicalExecutor) Add(task interface{}) {
	if vals, ok := pe.addAndCheck(task); ok {
		pe.commander <- vals
	}
}

func (pe *PeriodicalExecutor) ForceFlush() {
	pe.executeTasks(func() interface{} {
		pe.lock.Lock()
		defer pe.lock.Unlock()
		return pe.container.RemoveAll()
	}())
}

func (pe *PeriodicalExecutor) Sync(fn func()) {
	pe.lock.Lock()
	defer pe.lock.Unlock()
	fn()
}

func (pe *PeriodicalExecutor) addAndCheck(task interface{}) (interface{}, bool) {
	pe.lock.Lock()
	defer pe.lock.Unlock()

	if pe.container.AddTask(task) {
		return pe.container.RemoveAll(), true
	}

	return nil, false
}

func (pe *PeriodicalExecutor) backgroundFlush() {
	threading.GoSafe(func() {
		ticker := time.NewTicker(pe.interval)
		defer ticker.Stop()

		var commanded bool
		for {
			select {
			case vals := <-pe.commander:
				commanded = true
				pe.executeTasks(vals)
			case <-ticker.C:
				if commanded {
					commanded = false
				} else {
					pe.ForceFlush()
				}
			}
		}
	})
}

func (pe *PeriodicalExecutor) executeTasks(tasks interface{}) {
	if pe.hasTasks(tasks) {
		pe.container.Execute(tasks)
	}
}

func (pe *PeriodicalExecutor) hasTasks(tasks interface{}) bool {
	if tasks == nil {
		return false
	}

	val := reflect.ValueOf(tasks)
	switch val.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return val.Len() > 0
	default:
		// unknown type, let caller execute it
		return true
	}
}
