# go-panicguard

[![CI](https://github.com/philiprehberger/go-panicguard/actions/workflows/ci.yml/badge.svg)](https://github.com/philiprehberger/go-panicguard/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/philiprehberger/go-panicguard.svg)](https://pkg.go.dev/github.com/philiprehberger/go-panicguard) [![License](https://img.shields.io/github/license/philiprehberger/go-panicguard)](LICENSE)

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
| `Go(fn)` | Runs fn in a goroutine with panic recovery |
| `GoErr(fn)` | Runs fn in a goroutine, returns a channel with the error result or `*PanicError` |
| `Recover(r)` | Call in a deferred func with `recover()` to convert a panic into a `*PanicError` |
| `SetOnPanic(fn)` | Sets a global handler called on every recovered panic |
| `Middleware(next)` | HTTP middleware that recovers panics and returns 500 |
| `MiddlewareWithHandler(onPanic)` | HTTP middleware with a custom recovery response handler |

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT
