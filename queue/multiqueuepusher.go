package queue

import "lazygo/core/errorx"

type MultiQueuePusher struct {
	name    string
	pushers []QueuePusher
}

func NewMultiQueuePusher(pushers []QueuePusher) QueuePusher {
	return &MultiQueuePusher{
		name:    generateName(pushers),
		pushers: pushers,
	}
}

func (pusher *MultiQueuePusher) Name() string {
	return pusher.name
}

func (pusher *MultiQueuePusher) Push(message string) error {
	var batchError errorx.BatchError

	for _, each := range pusher.pushers {
		if err := each.Push(message); err != nil {
			batchError = append(batchError, err)
		}
	}

	return batchError
}
