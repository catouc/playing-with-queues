package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var workerCount int32

func main()  {
	q := make(chan string, 10)
	wg := sync.WaitGroup{}

	for i:=0; i < 1; i++ {
		wg.Add(1)
		atomic.AddInt32(&workerCount, 1)
		go worker(&wg, q)
	}

	go watchAndScale(context.Background(), QueueWatcher{
		Queue:         q,
		QueueWG:       &wg,
	})

	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		q <- fmt.Sprintf("message %d", i)
		fmt.Printf("Queuedepth: %d\n", len(q))
	}

	for i := 0; i < 100; i++ {
		time.Sleep(10 * time.Millisecond)
		q <- fmt.Sprintf("message %d", i)
		fmt.Printf("Queuedepth: %d\n", len(q))
	}

	close(q)

	wg.Wait()
}

func worker(wg *sync.WaitGroup, workChan chan string) {
	for s := range workChan {
		time.Sleep(1 * time.Second)
		fmt.Println(s)
	}
	wg.Done()
}

type QueueWatcher struct {
	Queue chan string
	QueueWG *sync.WaitGroup
	PreviousDepth int
	CurrentDepth int
}

func watchAndScale(ctx context.Context, watcher QueueWatcher) {
	ticker := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		case <- ctx.Done():
			fmt.Println(ctx.Err())
		case <- ticker.C:
			watcher.CurrentDepth = len(watcher.Queue)

			// < 0 means shrinking and > 0 means growing
			// we need to add workers when it's > 0
			growthTrend := watcher.CurrentDepth - watcher.PreviousDepth
			fmt.Printf("Growthtrend: %d\n", growthTrend)
			if growthTrend > 0 {
				watcher.QueueWG.Add(1)
				atomic.AddInt32(&workerCount, 1)
				go worker(watcher.QueueWG, watcher.Queue)
				fmt.Printf("added worker for a total of %d workers\n", workerCount)
			}
			if growthTrend < 0 {
				fmt.Println("would shrink workers by 1")
			}
			watcher.PreviousDepth = watcher.CurrentDepth
		}
	}
}
