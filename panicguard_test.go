package panicguard

import (
	"context"
	"errors"
	"fmt"
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

// --- GoNamed tests ---

func TestGoNamedRunsFunction(t *testing.T) {
	var mu sync.Mutex
	done := false
	var wg sync.WaitGroup
	wg.Add(1)

	GoNamed("test-task", func() {
		defer wg.Done()
		mu.Lock()
		done = true
		mu.Unlock()
	})

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if !done {
		t.Fatal("expected GoNamed to run the function")
	}
}

func TestGoNamedIncludesNameInPanic(t *testing.T) {
	var mu sync.Mutex
	var recovered any

	SetOnPanic(func(r any, s []byte) {
		mu.Lock()
		recovered = r
		mu.Unlock()
	})
	defer SetOnPanic(nil)

	GoNamed("worker-42", func() {
		panic("something broke")
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if recovered == nil {
		t.Fatal("expected OnPanic to be called")
	}
	s := fmt.Sprintf("%v", recovered)
	if !strings.Contains(s, "worker-42") {
		t.Errorf("expected recovered value to contain name %q, got %v", "worker-42", s)
	}
	if !strings.Contains(s, "something broke") {
		t.Errorf("expected recovered value to contain panic message, got %v", s)
	}
}

// --- GoCtx tests ---

func TestGoCtxRunsFunction(t *testing.T) {
	var mu sync.Mutex
	done := false
	var wg sync.WaitGroup
	wg.Add(1)

	GoCtx(context.Background(), func(ctx context.Context) {
		defer wg.Done()
		mu.Lock()
		done = true
		mu.Unlock()
	})

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if !done {
		t.Fatal("expected GoCtx to run the function")
	}
}

func TestGoCtxPassesContext(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "hello")

	var mu sync.Mutex
	var got any
	var wg sync.WaitGroup
	wg.Add(1)

	GoCtx(ctx, func(ctx context.Context) {
		defer wg.Done()
		mu.Lock()
		got = ctx.Value(key{})
		mu.Unlock()
	})

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if got != "hello" {
		t.Errorf("expected context value %q, got %v", "hello", got)
	}
}

func TestGoCtxRecoversPanic(t *testing.T) {
	var mu sync.Mutex
	var recovered any

	SetOnPanic(func(r any, s []byte) {
		mu.Lock()
		recovered = r
		mu.Unlock()
	})
	defer SetOnPanic(nil)

	GoCtx(context.Background(), func(ctx context.Context) {
		panic("ctx panic")
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if recovered == nil {
		t.Fatal("expected OnPanic to be called")
	}
	if recovered != "ctx panic" {
		t.Errorf("expected %q, got %v", "ctx panic", recovered)
	}
}

func TestGoCtxCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var mu sync.Mutex
	var ctxErr error
	var wg sync.WaitGroup
	wg.Add(1)

	GoCtx(ctx, func(ctx context.Context) {
		defer wg.Done()
		mu.Lock()
		ctxErr = ctx.Err()
		mu.Unlock()
	})

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if ctxErr != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", ctxErr)
	}
}

// --- Stats tests ---

func TestStatsIncrementOnPanic(t *testing.T) {
	ResetStats()
	SetOnPanic(nil)

	Go(func() { panic("stat-1") })
	Go(func() { panic("stat-2") })
	Go(func() { panic("stat-3") })

	time.Sleep(200 * time.Millisecond)

	stats := Stats()
	if stats.TotalPanics != 3 {
		t.Errorf("expected TotalPanics=3, got %d", stats.TotalPanics)
	}
	if stats.LastPanic.IsZero() {
		t.Error("expected LastPanic to be set")
	}
	if stats.LastValue == nil {
		t.Error("expected LastValue to be set")
	}
}

func TestStatsFromGoErr(t *testing.T) {
	ResetStats()
	SetOnPanic(nil)

	ch := GoErr(func() error {
		panic("goerr-stat")
	})
	<-ch

	stats := Stats()
	if stats.TotalPanics != 1 {
		t.Errorf("expected TotalPanics=1, got %d", stats.TotalPanics)
	}
}

func TestStatsFromGoNamed(t *testing.T) {
	ResetStats()
	SetOnPanic(nil)

	GoNamed("stat-named", func() { panic("named-stat") })
	time.Sleep(100 * time.Millisecond)

	stats := Stats()
	if stats.TotalPanics != 1 {
		t.Errorf("expected TotalPanics=1, got %d", stats.TotalPanics)
	}
}

func TestStatsFromGoCtx(t *testing.T) {
	ResetStats()
	SetOnPanic(nil)

	GoCtx(context.Background(), func(ctx context.Context) { panic("ctx-stat") })
	time.Sleep(100 * time.Millisecond)

	stats := Stats()
	if stats.TotalPanics != 1 {
		t.Errorf("expected TotalPanics=1, got %d", stats.TotalPanics)
	}
}

func TestResetStats(t *testing.T) {
	ResetStats()
	SetOnPanic(nil)

	Go(func() { panic("before-reset") })
	time.Sleep(100 * time.Millisecond)

	if Stats().TotalPanics != 1 {
		t.Fatal("expected 1 panic before reset")
	}

	ResetStats()
	stats := Stats()
	if stats.TotalPanics != 0 {
		t.Errorf("expected TotalPanics=0 after reset, got %d", stats.TotalPanics)
	}
	if !stats.LastPanic.IsZero() {
		t.Error("expected LastPanic to be zero after reset")
	}
	if stats.LastValue != nil {
		t.Errorf("expected LastValue to be nil after reset, got %v", stats.LastValue)
	}
}

func TestStatsNoPanic(t *testing.T) {
	ResetStats()
	var wg sync.WaitGroup
	wg.Add(1)
	Go(func() {
		defer wg.Done()
		// no panic
	})
	wg.Wait()

	stats := Stats()
	if stats.TotalPanics != 0 {
		t.Errorf("expected TotalPanics=0, got %d", stats.TotalPanics)
	}
}

// --- RecoverAs tests ---

type customError struct {
	Code int
}

func (e *customError) Error() string {
	return fmt.Sprintf("code: %d", e.Code)
}

func TestRecoverAsNil(t *testing.T) {
	val, ok := RecoverAs[*PanicError](nil)
	if ok {
		t.Error("expected ok=false for nil")
	}
	if val != nil {
		t.Errorf("expected zero value, got %v", val)
	}
}

func TestRecoverAsDirectMatch(t *testing.T) {
	original := &customError{Code: 42}
	val, ok := RecoverAs[*customError](original)
	if !ok {
		t.Fatal("expected ok=true for direct match")
	}
	if val.Code != 42 {
		t.Errorf("expected Code=42, got %d", val.Code)
	}
}

func TestRecoverAsPanicError(t *testing.T) {
	// When r is a string, RecoverAs wraps it in PanicError. We should
	// be able to extract *PanicError via errors.As.
	val, ok := RecoverAs[*PanicError]("some string")
	if !ok {
		t.Fatal("expected ok=true for string wrapped in PanicError")
	}
	if val.Value != "some string" {
		t.Errorf("expected Value=%q, got %v", "some string", val.Value)
	}
}

func TestRecoverAsNoMatch(t *testing.T) {
	// A string panic cannot be cast to *customError
	val, ok := RecoverAs[*customError]("not a custom error")
	if ok {
		t.Error("expected ok=false for non-matching type")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}
