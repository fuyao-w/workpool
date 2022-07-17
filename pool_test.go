package work_pool

import (
	"testing"
	"time"
)

func buildWorker(p *pool, closeCallBack func(*Worker)) (w *Worker) {

	w = &Worker{
		pool:          p,
		task:          make(chan task),
		closeCallBack: closeCallBack,
	}
	w.run()
	return
}

func TestPoolClose(t *testing.T) {

	cb := func(w *Worker) {
		t.Log("close", w.IdleBeginAt)
	}
	p := NewPool()
	p.restoreWorker(buildWorker(p, cb))
	p.restoreWorker(buildWorker(p, cb))
	p.restoreWorker(buildWorker(p, cb))
	p.restoreWorker(buildWorker(p, cb))
	p.restoreWorker(buildWorker(p, cb))
	t.Log(p.workPool.len())

	p.Close()
	time.Sleep(10 * time.Second)
}

func TestPool(t *testing.T) {
	var p = NewPool(WithWorkerCloseCallBack(func(worker *Worker) {
		t.Log("close ", worker.IdleBeginAt)
	}))
	for i := 0; i < 1000; i++ {
		i := i
		p.Submit(func() {
			t.Log(i)
		})
	}
	p.Close()
	time.Sleep(10 * time.Minute)
}

func TestBroadcast(t *testing.T) {
	var p = NewPool(
		WithWorkerCloseCallBack(func(worker *Worker) {
			t.Log("close ", worker.IdleBeginAt)
		}),
		WithCapacity(5),
		//WithTTl(5*time.Second),
	)

	for i := 0; i < 1000; i++ {
		i := i
		t.Log(p.Submit(func() {
			t.Log(i)
		}))
	}

	time.Sleep(time.Minute)
}
