package utils

import (
	"testing"
	"time"
)

var (
	now      = time.Now()
	evenPool = func() (wp WorkStack) {
		wp.push(&Worker{}, now.Add(time.Second))
		wp.push(&Worker{}, now.Add(2*time.Second))
		wp.push(&Worker{}, now.Add(3*time.Second))
		wp.push(&Worker{}, now.Add(4*time.Second))
		return wp
	}()
	oddPool = func() (wp WorkStack) {
		wp.push(&Worker{}, now.Add(time.Second))
		wp.push(&Worker{}, now.Add(2*time.Second))
		wp.push(&Worker{}, now.Add(3*time.Second))
		wp.push(&Worker{}, now.Add(4*time.Second))
		wp.push(&Worker{}, now.Add(5*time.Second))

		return wp
	}()
)

func TestWorkPool(t *testing.T) {
	wp := oddPool
	search := func(at time.Duration) {
		t.Log(wp.search(now.Add(at * time.Second)))
	}
	for i := 0; i < 8; i++ {
		search(time.Duration(i))
	}
}

func TestWorkPool2(t *testing.T) {
	wp := evenPool

	search := func(at time.Duration) {
		t.Log(wp.search(now.Add(at * time.Second)))
	}

	for i := 0; i < 8; i++ {
		search(time.Duration(i))
	}
}

func TestTruncate(t *testing.T) {
	items := oddPool.truncateExpiredItems(now.Add(10 * time.Second))
	for _, item := range items {
		t.Log(item.IdleBeginAt)
	}
}

func TestPop(t *testing.T) {
	w := oddPool.pop()
	t.Log(w.IdleBeginAt)
	w = oddPool.pop()
	t.Log(w.IdleBeginAt)
	t.Log(oddPool.len())

}
