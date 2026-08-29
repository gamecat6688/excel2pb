package works

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

var wg sync.WaitGroup
var failureMu sync.Mutex
var failures []any
var limiterMu sync.RWMutex
var limiter chan struct{}

// SetLimit 设置同时运行的任务上限。必须在提交任务前调用。
func SetLimit(limit int) {
	limiterMu.Lock()
	defer limiterMu.Unlock()
	if limit <= 0 {
		limiter = nil
		return
	}
	limiter = make(chan struct{}, limit)
}

func Wait() {
	wg.Wait()

	failureMu.Lock()
	batchFailures := failures
	failures = nil
	failureMu.Unlock()
	if len(batchFailures) > 0 {
		panic(fmt.Sprintf("worker failed: %v (failures: %d)", batchFailures[0], len(batchFailures)))
	}
}

// Go 函数用于并发执行子协程任务。
func Go(fn func()) {
	limiterMu.RLock()
	currentLimiter := limiter
	limiterMu.RUnlock()
	wg.Add(1)

	go func() {
		defer wg.Done()
		if currentLimiter != nil {
			currentLimiter <- struct{}{}
			defer func() { <-currentLimiter }()
		}
		defer func() {
			if r := recover(); r != nil {
				failureMu.Lock()
				failures = append(failures, r)
				failureMu.Unlock()
				slog.Error(fmt.Sprintf("Caught by common recover: %v\n%s", r, debug.Stack()))
			}
		}()
		fn()
	}()

}

// RecoverFunc 接受一个函数类型的参数 fn。这个函数的作用是在执行传入的函数 fn 时，捕获可能发生的 panic 异常，并记录错误信息。
func RecoverFunc(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error(fmt.Sprintf("Caught by common recover: %v\n%s", r, debug.Stack()))
		}
	}()

	fn()
}
