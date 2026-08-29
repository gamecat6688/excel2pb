package works

import (
	"sync/atomic"
	"testing"
)

func TestGoWaitAndPanicRecovery(t *testing.T) {
	var completed atomic.Int32
	Go(func() { completed.Add(1) })
	Go(func() { panic("expected test panic") })
	Go(func() { completed.Add(1) })
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("Wait must propagate a worker panic")
		}
		if completed.Load() != 2 {
			t.Fatalf("completed tasks = %d, want 2", completed.Load())
		}
	}()
	Wait()
}

func TestSetLimitBoundsConcurrentTasks(t *testing.T) {
	SetLimit(1)
	defer SetLimit(0)

	var running atomic.Int32
	var maximum atomic.Int32
	started := make(chan struct{}, 3)
	release := make(chan struct{}, 3)
	for range 3 {
		Go(func() {
			current := running.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			running.Add(-1)
		})
	}
	for range 3 {
		<-started
		if got := running.Load(); got != 1 {
			t.Fatalf("running tasks = %d, want 1", got)
		}
		release <- struct{}{}
	}
	Wait()
	if got := maximum.Load(); got != 1 {
		t.Fatalf("maximum concurrent tasks = %d, want 1", got)
	}
}
