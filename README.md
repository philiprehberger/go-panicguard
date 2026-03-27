# go-panicguard

[![CI](https://github.com/philiprehberger/go-panicguard/actions/workflows/ci.yml/badge.svg)](https://github.com/philiprehberger/go-panicguard/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/philiprehberger/go-panicguard.svg)](https://pkg.go.dev/github.com/philiprehberger/go-panicguard) [![License](https://img.shields.io/github/license/philiprehberger/go-panicguard)](LICENSE) [![Sponsor](https://img.shields.io/badge/sponsor-GitHub%20Sponsors-ec6cb9)](https://github.com/sponsors/philiprehberger)

Panic recovery utilities for Go — safe goroutines, panic-to-error conversion, and HTTP handler protection

## Installation

```bash
go get github.com/philiprehberger/go-panicguard
```

## Usage

```go
import "github.com/philiprehberger/go-panicguard"
```

### Safe goroutines

```go
panicguard.Go(func() {
    // If this panics, the panic is recovered and the OnPanic hook is called.
    riskyWork()
})
```

### Goroutine with error result

```go
ch := panicguard.GoErr(func() error {
    return doWork()
})
if err := <-ch; err != nil {
    // err is either the returned error or a *panicguard.PanicError
    log.Println(err)
}
```

### Deferred panic-to-error conversion

```go
func handler() (err error) {
    defer func() {
        if e := panicguard.Recover(recover()); e != nil {
            err = e
        }
    }()
    riskyOperation()
    return nil
}
```

### Named goroutines

```go
panicguard.GoNamed("email-sender", func() {
    // If this panics, the name "email-sender" is included in the panic log.
    sendEmails()
})
```

### Context-aware goroutines

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

panicguard.GoCtx(ctx, func(ctx context.Context) {
    // ctx is passed through; panics are still recovered.
    fetchData(ctx)
})
```

### Panic statistics

```go
panicguard.Go(func() { panic("oops") })
time.Sleep(100 * time.Millisecond)

stats := panicguard.Stats()
fmt.Println(stats.TotalPanics) // 1
fmt.Println(stats.LastPanic)   // time of the last panic
fmt.Println(stats.LastValue)   // "oops"

panicguard.ResetStats() // reset counters (useful in tests)
```

### Typed panic recovery

```go
defer func() {
    if pe, ok := panicguard.RecoverAs[*panicguard.PanicError](recover()); ok {
        log.Printf("panic value: %v", pe.Value)
    }
}()
```

### Global panic hook

```go
panicguard.SetOnPanic(func(recovered any, stack []byte) {
    log.Printf("panic recovered: %v\n%s", recovered, stack)
})
```

### HTTP middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/", handler)
http.ListenAndServe(":8080", panicguard.Middleware(mux))
```

### Custom recovery response

```go
middleware := panicguard.MiddlewareWithHandler(func(w http.ResponseWriter, r *http.Request, recovered any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    fmt.Fprintf(w, `{"error":"%v"}`, recovered)
})
http.ListenAndServe(":8080", middleware(mux))
```

## API

| Function / Type | Description |
|-----------------|-------------|
| `PanicError` | Wraps a recovered panic value with `Value` and `Stack` fields |
| `PanicError.Error()` | Returns `"panic: {value}"` |
| `PanicStats` | Holds `TotalPanics`, `LastPanic`, and `LastValue` fields |
| `Go(fn)` | Runs fn in a goroutine with panic recovery |
| `GoErr(fn)` | Runs fn in a goroutine, returns a channel with the error result or `*PanicError` |
| `GoNamed(name, fn)` | Runs fn in a named goroutine; name is included in panic reports |
| `GoCtx(ctx, fn)` | Runs fn in a goroutine, passing ctx; panics are recovered |
| `Recover(r)` | Call in a deferred func with `recover()` to convert a panic into a `*PanicError` |
| `RecoverAs[T](r)` | Typed panic recovery — extracts error type T from a recovered value |
| `SetOnPanic(fn)` | Sets a global handler called on every recovered panic |
| `Stats()` | Returns global panic statistics (thread-safe) |
| `ResetStats()` | Resets global panic statistics to zero values |
| `Middleware(next)` | HTTP middleware that recovers panics and returns 500 |
| `MiddlewareWithHandler(onPanic)` | HTTP middleware with a custom recovery response handler |

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT
