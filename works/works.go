package works

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

var wg sync.WaitGroup

func Wait() {
	wg.Wait()
}

// Go 函数用于并发执行子协程任务。
func Go(fn func()) {
	wg.Add(1)

	go func() {
		defer RecoverFunc(func() {
			wg.Done()
		})

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
