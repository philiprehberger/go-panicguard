package panicguard

import (
	"net/http"
	"runtime/debug"
)

// Middleware returns an http.Handler that recovers from panics in the next
// handler. When a panic occurs, it writes a 500 Internal Server Error response
// and calls the global OnPanic hook if one is set.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				recordPanic(rec)
				if handler := getOnPanic(); handler != nil {
					handler(rec, stack)
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// MiddlewareWithHandler returns HTTP middleware that recovers from panics and
// delegates the response to the provided onPanic function. This allows callers
// to customize the error response (e.g., returning JSON, logging, etc.).
func MiddlewareWithHandler(onPanic func(w http.ResponseWriter, r *http.Request, recovered any)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					recordPanic(rec)
					onPanic(w, r, rec)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
