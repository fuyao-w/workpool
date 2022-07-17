package utils

import (
	"time"
)

type (
	Worker struct {
		pool          *pool
		task          chan task
		IdleBeginAt   time.Time
		closeCallBack func(*Worker)
		panicHandler  PanicHandler
	}
	PanicHandler func(err interface{})
)

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
				if task == nil {
					return
				}
				task()
				if w.pool.restoreWorker(w) {
					return
				}
			}
		}
	}()
}
