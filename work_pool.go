package utils

import (
	"errors"
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
		status                    int32         // 池状态 [on off]
		closeSingle               chan struct{} // 池关闭通知
		runningSize, blockingSize int64         // runningSize 正在执行任务数量 ，blockingSize 阻塞任务刷领
		workPool                  WorkPool      // 协程池
		opt                       *PoolOption   // options 参数
		lock                      *sync.RWMutex
		workerCache               *sync.Pool
		cond                      *sync.Cond // 用于同步任务
	}

	task func() //任务
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
		opt  = loadOptions(opts)
		lock = &sync.RWMutex{}
		cond = sync.NewCond(lock)
	)

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
					panicHandler:  opt.panicHandler,
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

/*
	Submit 提交任务
	如果池已关闭则返回 ErrPoolClose
	如果提交任务被阻塞，在等待期间池关闭则会返回 ErrPoolClose
	如果任务为空则返回 ErrTaskIsNull
	如果池已满,并且非阻塞则返回 ErrWorkerSizeOverflow

*/
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

// retrieveWorker 获取可用的协程，如果不够会新创建
func (p *pool) retrieveWorker() (w *Worker, _ error) {
	genWorker := func() {
		w = p.workerCache.Get().(*Worker)
		w.run()
	}

	p.lock.Lock()

	// 池子里有直接返回
	w = p.workPool.pop()
	if w != nil {
		p.lock.Unlock()
		return
	}

	switch {
	case p.getRunning() < p.opt.capacity: //池子没有，但是还没到容量则创建
		p.lock.Unlock()
		genWorker()
	default: //否则等待
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
			// fast fail
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

/*
	Close 立即关闭池，当前未执行的任务会直接返回失败
*/
func (p *pool) Close() {
	if !atomic.CompareAndSwapInt32(&p.status, on, off) {
		return
	}
	close(p.closeSingle)
	p.closeWorker(0)
	p.cond.Broadcast() //通知等待的协程立即结束任务
}

func (p *pool) getStatus() int32 {
	return atomic.LoadInt32(&p.status)
}

/*
	closeWorker 清理池中的协程
	expireAt time.Duration 该时间以及之前就在池中的协程将会被清理
*/
func (p *pool) closeWorker(expireAt time.Duration) {

	p.lock.Lock()
	items := p.workPool.truncateExpiredItems(time.Now().Add(expireAt))
	p.lock.Unlock()

	for _, item := range items {
		item.task <- nil
	}
}

// purgeWorker 协程池清理函数，定期清理达到最大空闲时间的任务，当池关闭的时候会直接返回，剩下的任务由 Close 函数处理
func (p *pool) purgeWorker() {
	sticker := time.NewTicker(p.opt.ttl)
	defer sticker.Stop()
	for {
		select {
		case <-sticker.C:
			p.closeWorker(-p.opt.ttl)
			if p.getRunning() == 0 {
				p.cond.Broadcast() //通知等待的任务进行下一步处理
			}
		case <-p.closeSingle:
			return
		}
	}
}

/*
	restoreWorker 向池归还工作协程，并通知其他阻塞的任务
	w *Worker  待归还的协程
	return 当前协程不用归还池的时候返回 true ,否则返回false
*/
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
