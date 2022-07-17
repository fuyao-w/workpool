package utils

import (
	"log"
	"math"
	"runtime"
	"time"
)

type (
	PoolOption struct {
		ttl                        time.Duration
		nonBlocking                bool
		capacity, maxBlockingTasks int64
		workerCloseCallBack        func(*Worker)
		panicHandler               PanicHandler
		logger                     Logger
	}
	OptionF func(opt *PoolOption)
	Logger  interface {
		Printf(format string, v ...interface{})
	}
)

func WithTTl(ttl time.Duration) OptionF {
	return func(opt *PoolOption) {
		opt.ttl = ttl
	}
}
func WithLogger(logger Logger) OptionF {
	return func(opt *PoolOption) {
		opt.logger = logger
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

func WithPanicHandler(handler PanicHandler) OptionF {
	return func(opt *PoolOption) {
		opt.panicHandler = handler
	}
}

func loadOptions(opts []OptionF) (opt PoolOption) {
	for _, f := range opts {
		f(&opt)
	}
	if opt.ttl <= 0 {
		opt.ttl = time.Second
	}
	if opt.capacity <= 0 {
		opt.capacity = math.MaxInt64
	}
	if opt.logger == nil {
		opt.logger = log.Default()
	}

	if opt.panicHandler == nil {
		opt.panicHandler = func(err interface{}) {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			opt.logger.Printf("worker exits from panic ,err: %v ,stack: %s\n", err, string(buf[:n]))
		}
	}
	return
}
