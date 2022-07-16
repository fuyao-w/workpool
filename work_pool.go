package utils

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	off = 0
	on  = 1
)

var (
	ErrPoolClose          = errors.New("already close")
	ErrTaskIsNull         = errors.New("task is null")
	ErrWorkerSizeOverflow = errors.New("worker size overflow")
)

type (
	pool struct {
		status                    int32
		closeSingle               chan struct{}
		runningSize, blockingSize int64
		workPool                  WorkPool
		opt                       *PoolOption
		lock                      *sync.RWMutex
		workerCache               *sync.Pool
		cond                      *sync.Cond
	}

	task func()
)

func (p *pool) incrRunning() {
	atomic.AddInt64(&p.runningSize, 1)
}
func (p *pool) decrRunning() {
	atomic.AddInt64(&p.runningSize, -1)
}
func (p *pool) getRunning() int64 {
	return atomic.LoadInt64(&p.runningSize)
}

func (p *pool) incrBlocking() {
	atomic.AddInt64(&p.blockingSize, 1)
}
func (p *pool) decrBlocking() {
	atomic.AddInt64(&p.blockingSize, -1)
}
func (p *pool) getBlocking() int64 {
	return atomic.LoadInt64(&p.blockingSize)
}

func NewPool(opts ...OptionF) (p *pool) {
	var (
		opt  PoolOption
		lock = &sync.RWMutex{}
		cond = sync.NewCond(lock)
	)
	for _, f := range opts {
		f(&opt)
	}
	if opt.ttl <= 0 {
		opt.ttl = time.Second
	}
	if opt.capacity <= 0 {
		opt.capacity = math.MaxInt64
	}

	p = &pool{
		status:      on,
		closeSingle: make(chan struct{}),
		opt:         &opt,
		lock:        lock,
		cond:        cond,
		workPool:    NewWorkerPool(),
		workerCache: &sync.Pool{
			New: func() interface{} {
				worker := &Worker{
					closeCallBack: opt.workerCloseCallBack,
					task:          make(chan task, 1),
					pool:          p,
					IdleBeginAt:   time.Now(),
				}
				return worker
			},
		},
	}
	go p.purgeWorker()
	return
}

func (p *pool) Submit(task task) error {
	if task == nil {
		return ErrTaskIsNull
	}
	if p.getStatus() == off {
		return ErrPoolClose
	}
	w, err := p.retrieveWorker()
	if err != nil {
		return err
	}
	w.task <- task
	return nil
}

func (p *pool) retrieveWorker() (w *Worker, _ error) {
	genWorker := func() {
		w = p.workerCache.Get().(*Worker)
		w.run()
	}

	p.lock.Lock()

	w = p.workPool.pop()
	if w != nil {
		p.lock.Unlock()
		return
	}
	switch {
	case p.getRunning() < p.opt.capacity:
		p.lock.Unlock()
		genWorker()
	default:
		if p.opt.nonBlocking {
			p.lock.Unlock()
			return nil, ErrWorkerSizeOverflow
		}
		for {
			if p.opt.maxBlockingTasks > 0 && p.blockingSize >= p.opt.maxBlockingTasks {
				p.lock.Unlock()
				return nil, ErrWorkerSizeOverflow
			}
			p.incrBlocking()
			p.cond.Wait()
			p.decrBlocking()
			if p.getStatus() == off {
				p.lock.Unlock()
				return nil, ErrPoolClose
			}
			if p.getRunning() == 0 {
				p.lock.Unlock()
				genWorker()
				return
			}
			if w = p.workPool.pop(); w != nil {
				p.lock.Unlock()
				return
			}

		}
	}

	return
}
func (p *pool) Close() {
	if !atomic.CompareAndSwapInt32(&p.status, on, off) {
		return
	}
	close(p.closeSingle)
	p.closeWorker(0)
	p.cond.Broadcast()
}

func (p *pool) getStatus() int32 {
	return atomic.LoadInt32(&p.status)
}

func (p *pool) closeWorker(expireAt time.Duration) {

	p.lock.Lock()
	items := p.workPool.truncateExpiredItems(time.Now().Add(expireAt))
	p.lock.Unlock()

	for _, item := range items {
		item.task <- nil
	}
}

func (p *pool) purgeWorker() {
	sticker := time.NewTicker(p.opt.ttl)
	defer sticker.Stop()
	for {
		select {
		case <-sticker.C:
			p.closeWorker(-p.opt.ttl)
			if p.getRunning() == 0 {
				p.cond.Broadcast()
			}
		case <-p.closeSingle:
			return
		}
	}
}

func (p *pool) restoreWorker(w *Worker) bool {
	if p.getStatus() == off || p.getRunning() > p.opt.capacity {
		return true
	}

	p.lock.Lock()
	p.workPool.push(w)
	p.lock.Unlock()
	p.cond.Signal()
	return false
}

func NewWorkerPool() WorkPool {
	return &WorkStack{}
}
