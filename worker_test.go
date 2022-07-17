package work_pool

import (
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	p := NewPool()
	w := Worker{
		pool:        p,
		task:        make(chan task),
		IdleBeginAt: time.Now(),
		closeCallBack: func(worker *Worker) {
			t.Log("close", worker.IdleBeginAt)
		},
	}
	w.run()
	for i := 0; i < 3; i++ {
		i := i
		w.task <- func() {
			t.Log(i)
		}
	}
	w.task <- nil
	time.Sleep(time.Second)
}
