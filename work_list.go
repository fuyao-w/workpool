package utils

import (
	"time"
)

type (
	WorkPool interface {
		push(worker *Worker, mockBeginAt ...time.Time)
		pop() *Worker
		len() int
		truncateExpiredItems(endTime time.Time) []*Worker
	}
	WorkStack struct {
		items []*Worker
	}
)

func (w *WorkStack) push(worker *Worker, mockBeginAt ...time.Time) {
	if len(mockBeginAt) != 0 {
		worker.IdleBeginAt = mockBeginAt[0]
	} else {
		t := time.Now()
		worker.IdleBeginAt = t
	}
	w.items = append(w.items, worker)
}
func (w *WorkStack) pop() (worker *Worker) {
	if w.len() == 0 {
		return
	}
	worker = w.items[len(w.items)-1]
	w.items = w.items[:len(w.items)-1]
	return
}

func (w *WorkStack) len() int {
	return len(w.items)
}

func (w *WorkStack) search(expiredAt time.Time) int {
	var begin, end = 0, len(w.items) - 1
	for begin <= end {
		mid := begin>>1 + end>>1
		if begin%2 == 1 && end%2 == 1 {
			mid++
		}

		if expiredAt.Before(w.items[mid].IdleBeginAt) {
			end = mid - 1
		} else {
			begin = mid + 1
		}

	}
	return end + 1
}

func (w *WorkStack) truncateExpiredItems(endTime time.Time) []*Worker {
	idx := w.search(endTime)
	if idx == 0 {
		return nil
	}
	expireItems := w.items[:idx]
	w.items = append([]*Worker(nil), w.items[idx:]...)
	return expireItems
}
