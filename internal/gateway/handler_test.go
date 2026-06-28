package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var errNotReadyForTest = errors.New("not ready")

func TestHealthz(t *testing.T) {
	mux := NewHealthMux()
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok newline", recorder.Body.String())
	}
}

func TestReadyzWithoutChecks(t *testing.T) {
	mux := NewHealthMux()
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok newline", recorder.Body.String())
	}
}

func TestReadyzFailsWhenDependencyCheckFails(t *testing.T) {
	mux := NewHealthMux(func(_ context.Context) error {
		return errNotReadyForTest
	})
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
