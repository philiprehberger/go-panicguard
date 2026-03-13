package panicguard

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGoRunsFunction(t *testing.T) {
	var mu sync.Mutex
	done := false
	var wg sync.WaitGroup
	wg.Add(1)

	Go(func() {
		defer wg.Done()
		mu.Lock()
		done = true
		mu.Unlock()
	})

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if !done {
		t.Fatal("expected Go to run the function")
	}
}

func TestGoRecoversPanic(t *testing.T) {
	// Set up a hook to verify the panic was caught
	var mu sync.Mutex
	var recovered any
	var stack []byte

	SetOnPanic(func(r any, s []byte) {
		mu.Lock()
		recovered = r
		stack = s
		mu.Unlock()
	})
	defer SetOnPanic(nil)

	Go(func() {
		panic("test panic")
	})

	// Wait briefly for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if recovered == nil {
		t.Fatal("expected OnPanic hook to be called")
	}
	if recovered != "test panic" {
		t.Errorf("expected recovered value %q, got %v", "test panic", recovered)
	}
	if len(stack) == 0 {
		t.Error("expected non-empty stack trace")
	}
}

func TestGoErrWithSuccess(t *testing.T) {
	ch := GoErr(func() error {
		return nil
	})

	err := <-ch
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestGoErrWithError(t *testing.T) {
	want := errors.New("something failed")
	ch := GoErr(func() error {
		return want
	})

	err := <-ch
	if err != want {
		t.Errorf("expected %v, got %v", want, err)
	}
}

func TestGoErrWithPanic(t *testing.T) {
	SetOnPanic(nil)

	ch := GoErr(func() error {
		panic("boom")
	})

	err := <-ch
	if err == nil {
		t.Fatal("expected error from panicking goroutine")
	}

	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PanicError, got %T: %v", err, err)
	}
	if pe.Value != "boom" {
		t.Errorf("expected panic value %q, got %v", "boom", pe.Value)
	}
	if len(pe.Stack) == 0 {
		t.Error("expected non-empty stack in PanicError")
	}
}

func TestGoErrChannelClosed(t *testing.T) {
	ch := GoErr(func() error {
		return nil
	})

	// Drain the result
	<-ch

	// Channel should be closed; second receive should return zero value immediately
	err, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}
	if err != nil {
		t.Errorf("expected nil from closed channel, got %v", err)
	}
}

func TestRecoverWithPanic(t *testing.T) {
	var caught error
	func() {
		defer func() {
			caught = Recover(recover())
		}()
		panic("deferred panic")
	}()

	if caught == nil {
		t.Fatal("expected Recover to return an error")
	}

	var pe *PanicError
	if !errors.As(caught, &pe) {
		t.Fatalf("expected *PanicError, got %T", caught)
	}
	if pe.Value != "deferred panic" {
		t.Errorf("expected panic value %q, got %v", "deferred panic", pe.Value)
	}
	if len(pe.Stack) == 0 {
		t.Error("expected non-empty stack")
	}
}

func TestRecoverWithoutPanic(t *testing.T) {
	var caught error
	func() {
		defer func() {
			caught = Recover(recover())
		}()
		// no panic
	}()

	if caught != nil {
		t.Errorf("expected nil from Recover when no panic, got %v", caught)
	}
}

func TestPanicErrorFormat(t *testing.T) {
	pe := &PanicError{Value: "oops", Stack: []byte("fake stack")}
	want := "panic: oops"
	if pe.Error() != want {
		t.Errorf("got %q, want %q", pe.Error(), want)
	}
}

func TestPanicErrorFormatInt(t *testing.T) {
	pe := &PanicError{Value: 42, Stack: []byte("fake stack")}
	want := "panic: 42"
	if pe.Error() != want {
		t.Errorf("got %q, want %q", pe.Error(), want)
	}
}

func TestSetOnPanicReceivesValueAndStack(t *testing.T) {
	var mu sync.Mutex
	var gotValue any
	var gotStack []byte

	SetOnPanic(func(recovered any, stack []byte) {
		mu.Lock()
		gotValue = recovered
		gotStack = stack
		mu.Unlock()
	})
	defer SetOnPanic(nil)

	Go(func() {
		panic("hook test")
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if gotValue != "hook test" {
		t.Errorf("expected hook to receive %q, got %v", "hook test", gotValue)
	}
	if len(gotStack) == 0 {
		t.Error("expected non-empty stack in hook")
	}
	if !strings.Contains(string(gotStack), "goroutine") {
		t.Errorf("expected stack to contain goroutine info, got %q", string(gotStack))
	}
}

func TestSetOnPanicClear(t *testing.T) {
	called := false
	SetOnPanic(func(any, []byte) {
		called = true
	})
	SetOnPanic(nil)

	Go(func() {
		panic("should not call handler")
	})

	time.Sleep(100 * time.Millisecond)

	if called {
		t.Error("expected handler not to be called after clearing")
	}
}
