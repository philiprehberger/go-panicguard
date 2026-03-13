package panicguard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareNormalHandler(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestMiddlewarePanicHandler(t *testing.T) {
	SetOnPanic(nil)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("handler panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestMiddlewarePanicCallsOnPanic(t *testing.T) {
	var recovered any
	SetOnPanic(func(r any, s []byte) {
		recovered = r
	})
	defer SetOnPanic(nil)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("hook check")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if recovered != "hook check" {
		t.Errorf("expected OnPanic to receive %q, got %v", "hook check", recovered)
	}
}

func TestMiddlewareWithHandlerCustomResponse(t *testing.T) {
	middleware := MiddlewareWithHandler(func(w http.ResponseWriter, r *http.Request, recovered any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"%v"}`, recovered)
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("custom panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	want := `{"error":"custom panic"}`
	if rec.Body.String() != want {
		t.Errorf("expected body %q, got %q", want, rec.Body.String())
	}
}

func TestMiddlewareWithHandlerNormalRequest(t *testing.T) {
	middleware := MiddlewareWithHandler(func(w http.ResponseWriter, r *http.Request, recovered any) {
		t.Error("onPanic should not be called for normal requests")
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
