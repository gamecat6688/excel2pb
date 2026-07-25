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
	Wait()
	if completed.Load() != 2 {
		t.Fatalf("completed tasks = %d, want 2", completed.Load())
	}
}
