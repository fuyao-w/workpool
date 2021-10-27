package utils

import (
	"context"
	"fmt"
	"go.uber.org/atomic"
	"runtime"
	"sync"
	"time"
)

type (
	run            func(ctx context.Context)
	timeOutChecker struct {
		checker chan struct{}
		timeOut int64
	}
	pool struct {
		timeOutChecker timeOutChecker
		size, cur      int
		status         atomic.Int32
		workerList     chan run
		lock           sync.Mutex
	}
)

const (
	running int32 = iota + 1
	shutdown
)

func (p *pool) check() {
	fmt.Println("begin check", len(p.timeOutChecker.checker))
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			p.lock.Lock()
			if p.cur == 0 {
				p.lock.Unlock()
				return
			}
			p.lock.Unlock()

			if len(p.timeOutChecker.checker) == 0 {
				fmt.Println("check ----")
				p.timeOutChecker.checker <- struct{}{}
			}

		}
	}
}

func NewPool(n int, timeOut int64) (p *pool) {
	return &pool{
		timeOutChecker: timeOutChecker{
			checker: make(chan struct{}, 1),
			timeOut: timeOut,
		},

		size:       n,
		workerList: make(chan run, 20),
	}
}
func (p *pool) ShutDown() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.status.Store(shutdown)
}

func (p *pool) AddTask(ctx context.Context, task run) {
	if p.status.Load() == shutdown {
		return
	}
	if p.cur >= p.size {
		p.workerList <- task
		return
	}
	p.lock.Lock()
	go func(id int) {
		lastActiveTime := time.Now().Unix()
		for {
			select {
			case run := <-p.workerList:
				func() {
					defer func() {
						if p := recover(); p != nil {
							buf := make([]byte, 64<<10)
							buf = buf[:runtime.Stack(buf, false)]
							fmt.Println("panic !!", string(buf))
						}
					}()
					fmt.Println("do task ", id, lastActiveTime)
					run(ctx)
				}()
			case <-p.timeOutChecker.checker:
				fmt.Println("consume", len(p.timeOutChecker.checker))
				if (p.status.Load() == shutdown && len(p.workerList) == 0) || (p.timeOutChecker.timeOut != 0 && time.Now().Unix()-lastActiveTime >= p.timeOutChecker.timeOut) {
					p.lock.Lock()
					p.cur--
					p.lock.Unlock()
					fmt.Println("task stop", id)
					return
				}
			}
			lastActiveTime = time.Now().Unix()
		}
	}(p.cur)
	p.cur++
	p.workerList <- task
	if p.cur == 1 {
		p.status.Store(running)
		go p.check()
	}
	p.lock.Unlock()

}
