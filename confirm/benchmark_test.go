package confirm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkInFunc(b *testing.B) {
	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		innerWorker := func(ctx context.Context, num int) {
			num++
		}
		innerWorker(ctx, n)
	}
}

//go:noinline
func outerWorker(ctx context.Context, num int) {
	num++
}

func BenchmarkOutFunc(b *testing.B) {
	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		outerWorker(ctx, n)
	}
}

func BenchmarkConcurrency(b *testing.B) {
	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		innerWorker := func(ctx context.Context, num int) {
			num++
		}
		go innerWorker(ctx, n)
	}
}

func BenchmarkTimer(b *testing.B) {
	var (
		interval = 1 * time.Millisecond
	)
	for n := 0; n < b.N; n++ {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			counter     int
		)
		timer := time.NewTicker(interval)

		for {
			select {
			case <-ctx.Done():
				goto GOTO_FINISH
			case <-timer.C:
				counter++
				if counter >= 50 {
					cancel()
				}
			}
		}

	GOTO_FINISH:
		timer.Stop()
	}
}

func BenchmarkSleep(b *testing.B) {
	var (
		interval = 1 * time.Millisecond
	)
	for n := 0; n < b.N; n++ {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			counter     int
		)

		for {
			time.Sleep(interval)

			select {
			case <-ctx.Done():
				goto GOTO_FINISH
			default:
				counter++
				if counter >= 50 {
					cancel()
				}
			}
		}
	GOTO_FINISH:
	}
}

func BenchmarkAtomic(b *testing.B) {
	var (
		interval = 1 * time.Millisecond
	)
	for n := 0; n < b.N; n++ {
		var (
			counter uint32
		)
		timer := time.NewTicker(interval)

		for {
			select {
			case <-timer.C:
				atomic.AddUint32(&counter, 1)
				if atomic.LoadUint32(&counter) >= uint32(50) {
					goto GOTO_FINISH
				}
			}
		}
	GOTO_FINISH:
		timer.Stop()
	}
}
