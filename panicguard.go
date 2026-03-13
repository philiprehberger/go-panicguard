// Package panicguard provides panic recovery utilities for Go.
package panicguard

import (
	"fmt"
	"runtime/debug"
	"sync"
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
// recovered by Go, GoErr, or the HTTP middleware. The handler receives the
// recovered value and the stack trace captured at the point of the panic.
// Pass nil to clear the handler.
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
