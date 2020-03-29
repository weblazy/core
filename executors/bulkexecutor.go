package executors

import "time"

const (
	defaultCachedTasks   = 1000
	defaultFlushInterval = time.Second
)

type (
	BulkExecutorOption func(options *ExecutorOptions)
	Execute            func(tasks []interface{})

	BulkExecutor struct {
		executor  *PeriodicalExecutor
		container *bulkContainer
	}

	ExecutorOptions struct {
		CachedTasks   int
		FlushInterval time.Duration
	}
)

func NewBulkExecutor(execute Execute, opts ...BulkExecutorOption) *BulkExecutor {
	options := newExecutorOptions()
	for _, opt := range opts {
		opt(&options)
	}

	container := &bulkContainer{
		execute:  execute,
		maxTasks: options.CachedTasks,
	}
	executor := &BulkExecutor{
		executor:  NewPeriodicalExecutor(options.FlushInterval, container),
		container: container,
	}

	return executor
}

func (bi *BulkExecutor) Flush() {
	bi.executor.ForceFlush()
}

func (bi *BulkExecutor) Add(task interface{}) error {
	bi.executor.Add(task)
	return nil
}

func CacheTasks(tasks int) BulkExecutorOption {
	return func(options *ExecutorOptions) {
		options.CachedTasks = tasks
	}
}

func FlushInterval(duration time.Duration) BulkExecutorOption {
	return func(options *ExecutorOptions) {
		options.FlushInterval = duration
	}
}

func newExecutorOptions() ExecutorOptions {
	return ExecutorOptions{
		CachedTasks:   defaultCachedTasks,
		FlushInterval: defaultFlushInterval,
	}
}

type bulkContainer struct {
	tasks    []interface{}
	execute  Execute
	maxTasks int
}

func (bc *bulkContainer) AddTask(task interface{}) bool {
	bc.tasks = append(bc.tasks, task)
	return len(bc.tasks) >= bc.maxTasks
}

func (bc *bulkContainer) Execute(bulk interface{}) {
	tasks := bulk.([]interface{})
	bc.execute(tasks)
}

func (bc *bulkContainer) RemoveAll() interface{} {
	tasks := bc.tasks
	bc.tasks = nil
	return tasks
}
