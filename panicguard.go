// Package panicguard provides panic recovery utilities for Go.
package panicguard

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// PanicError wraps a recovered panic value along with the stack trace
// captured at the point of the panic.
type PanicError struct {
	// Value is the value that was passed to panic().
	Value any
	// Stack is the raw stack trace captured via runtime/debug.Stack().
	Stack []byte
}

// Error returns a string representation of the panic in the format "panic: {value}".
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}

var (
	onPanic   func(any, []byte)
	onPanicMu sync.RWMutex
)

// SetOnPanic sets a global panic handler that is called whenever a panic is
// recovered by Go, GoErr, GoNamed, GoCtx, or the HTTP middleware. The handler
// receives the recovered value and the stack trace captured at the point of
// the panic. Pass nil to clear the handler.
func SetOnPanic(fn func(recovered any, stack []byte)) {
	onPanicMu.Lock()
	defer onPanicMu.Unlock()
	onPanic = fn
}

// getOnPanic returns the current global panic handler, if any.
func getOnPanic() func(any, []byte) {
	onPanicMu.RLock()
	defer onPanicMu.RUnlock()
	return onPanic
}

// Go runs fn in a new goroutine with panic recovery. If fn panics, the panic
// is recovered and the global OnPanic handler is called (if set). The panic
// is not propagated.
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				recordPanic(r)
				if handler := getOnPanic(); handler != nil {
					handler(r, stack)
				}
			}
		}()
		fn()
	}()
}

// GoErr runs fn in a new goroutine and returns a buffered channel that will
// receive the result. If fn returns an error, it is sent on the channel. If
// fn panics, a *PanicError is sent instead. The channel is closed after the
// value is sent.
func GoErr(fn func() error) <-chan error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				recordPanic(r)
				if handler := getOnPanic(); handler != nil {
					handler(r, stack)
				}
				ch <- &PanicError{Value: r, Stack: stack}
			}
		}()
		ch <- fn()
	}()
	return ch
}

// Recover converts a recovered panic value into a *PanicError. It is meant to
// be used inside a deferred function alongside the built-in recover():
//
//	defer func() {
//	    if err := panicguard.Recover(recover()); err != nil {
//	        // handle err (*PanicError)
//	    }
//	}()
//
// Returns nil if r is nil (no panic occurred).
func Recover(r any) error {
	if r != nil {
		return &PanicError{
			Value: r,
			Stack: debug.Stack(),
		}
	}
	return nil
}

// GoNamed runs fn in a new goroutine with panic recovery. The name is included
// in the PanicError and passed to the global OnPanic hook for debugging.
func GoNamed(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				recordPanic(r)
				if handler := getOnPanic(); handler != nil {
					handler(fmt.Sprintf("[%s] %v", name, r), stack)
				}
			}
		}()
		fn()
	}()
}

// GoCtx runs fn in a new goroutine with panic recovery, passing ctx to fn.
// If ctx is already cancelled before fn panics, the panic is still recovered.
func GoCtx(ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				recordPanic(r)
				if handler := getOnPanic(); handler != nil {
					handler(r, stack)
				}
			}
		}()
		fn(ctx)
	}()
}

// PanicStats holds global panic statistics.
type PanicStats struct {
	// TotalPanics is the total number of panics recovered across all functions.
	TotalPanics int64
	// LastPanic is the time of the most recent recovered panic.
	LastPanic time.Time
	// LastValue is the value from the most recent recovered panic.
	LastValue any
}

var (
	statsTotalPanics atomic.Int64
	statsLastPanic   atomic.Value // stores time.Time
	statsLastValue   atomic.Value // stores any
)

// recordPanic updates the global panic statistics. It is called from all
// recovery paths (Go, GoErr, GoNamed, GoCtx).
func recordPanic(value any) {
	statsTotalPanics.Add(1)
	statsLastPanic.Store(time.Now())
	statsLastValue.Store(value)
}

// Stats returns the current global panic statistics.
func Stats() PanicStats {
	var lastPanic time.Time
	if v := statsLastPanic.Load(); v != nil {
		lastPanic = v.(time.Time)
	}
	var lastValue any
	if v := statsLastValue.Load(); v != nil {
		lastValue = v
	}
	return PanicStats{
		TotalPanics: statsTotalPanics.Load(),
		LastPanic:   lastPanic,
		LastValue:   lastValue,
	}
}

// ResetStats resets the global panic statistics to zero values. This is useful
// for testing.
func ResetStats() {
	statsTotalPanics.Store(0)
	statsLastPanic = atomic.Value{}
	statsLastValue = atomic.Value{}
}

// RecoverAs attempts to extract a typed error from a recovered panic value.
// If r is nil, it returns the zero value of T and false. If r implements T
// directly, it returns the value and true. Otherwise, r is wrapped in a
// PanicError and errors.As is used to attempt the conversion.
func RecoverAs[T error](r any) (T, bool) {
	var zero T
	if r == nil {
		return zero, false
	}
	// Direct type assertion
	if t, ok := r.(T); ok {
		return t, true
	}
	// Wrap in PanicError and try errors.As
	wrapped := &PanicError{Value: r, Stack: debug.Stack()}
	var target T
	if errors.As(wrapped, &target) {
		return target, true
	}
	return zero, false
}
