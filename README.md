
# 协程池

用法
```go
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
```

