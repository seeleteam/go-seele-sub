package common

import (
	"fmt"
	"runtime"
)

var funcCallerOrder int

func Trace() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	funcCallerOrder++
	fmt.Printf("[TEST trace %d]%s:%d %s\n", funcCallerOrder, file, line, f.Name())
}

func Trace2() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	// file, line := f.FileLine(pc[0])
	funcCallerOrder++
	fmt.Printf("[TEST trace %d] %s\n", funcCallerOrder, f.Name())
}
