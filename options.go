package utils

import "time"

type (
	PoolOption struct {
		ttl                        time.Duration
		nonBlocking                bool
		capacity, maxBlockingTasks int64
		workerCloseCallBack        func(*Worker)
	}
	OptionF func(opt *PoolOption)
)

func WithTTl(ttl time.Duration) OptionF {
	return func(opt *PoolOption) {
		opt.ttl = ttl
	}
}
func WithNonBlocking(nonBlocking bool) OptionF {
	return func(opt *PoolOption) {
		opt.nonBlocking = nonBlocking
	}
}
func WithCapacity(capacity int64) OptionF {
	return func(opt *PoolOption) {
		opt.capacity = capacity
	}
}
func WithMaxBlockingTasks(maxBlockingTasks int64) OptionF {
	return func(opt *PoolOption) {
		opt.maxBlockingTasks = maxBlockingTasks
	}
}

func WithWorkerCloseCallBack(cb func(worker *Worker)) OptionF {
	return func(opt *PoolOption) {
		opt.workerCloseCallBack = cb
	}
}
