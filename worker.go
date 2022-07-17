package utils

import (
	"time"
)

type (
	Worker struct { //协程
		pool          *pool //池反向引用
		task          chan task
		IdleBeginAt   time.Time //协程上次空闲的开始时间
		closeCallBack func(*Worker)
		panicHandler  PanicHandler
	}
	PanicHandler func(err interface{})
)

// run 运行协程，开始接受任务
func (w *Worker) run() {
	w.pool.incrRunning()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				if w.panicHandler != nil {
					w.panicHandler(p)
				}
			}
			w.pool.decrRunning()
			w.pool.workerCache.Put(w)
			if w.closeCallBack != nil {
				w.closeCallBack(w)
			}
		}()

		for {
			select {
			case task := <-w.task:
				if task == nil { //关闭信号
					return
				}
				task()
				if w.pool.restoreWorker(w) { //如果任务不需要被归还给池，则直接结束
					return
				}
			}
		}
	}()
}
