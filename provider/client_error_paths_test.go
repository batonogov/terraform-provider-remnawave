package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestServer5xxReturnsError verifies that a 5xx response from the backend is
// surfaced as a non-nil *HTTPStatusError and is NOT retried by the client.
//
// The HTTP client in this provider intentionally does not retry 5xx responses
// — the caller (Terraform) is responsible for surfacing the error to the user.
// This test pins that contract so a future "auto-retry" change does not slip
// in silently.
//
// Covers #118 (5xx retry behaviour — documents no-retry contract).
func TestServer5xxReturnsError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"message":"upstream down"}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "token"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("expected error for 502, got nil")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want *HTTPStatusError", err)
	}
	if statusErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", statusErr.StatusCode)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server call count = %d, want exactly 1 (no retry)", got)
	}
}

// TestServer503SurfacesError verifies a 503 response is surfaced as an error.
// Together with TestServer5xxReturnsError this documents the contract for the
// full 5xx range that matters for reliability (502 from reverse proxy, 503
// when backend is starting, 504 gateway timeout shape via transport error).
func TestServer503SurfacesError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "token"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("expected error for 503, got nil")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want *HTTPStatusError", err)
	}
	if statusErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", statusErr.StatusCode)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server call count = %d, want exactly 1 (no retry)", got)
	}
}

// TestServer403ForbiddenSurfacesError verifies that a 403 Forbidden response
// is surfaced as an *HTTPStatusError and is not retried. This is the error
// shape resources receive when the current auth context lacks permission for
// the endpoint (e.g. passkeys under a scoped API token).
//
// Covers #118 (403 Forbidden).
func TestServer403ForbiddenSurfacesError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "token"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want *HTTPStatusError", err)
	}
	if statusErr.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", statusErr.StatusCode)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server call count = %d, want exactly 1 (no retry on 403)", got)
	}
}

// TestServer404SurfacesNotFoundError verifies that a 404 response surfaces as
// an *HTTPStatusError whose message matches the pattern that isNotFound()
// recognises. This pins the bridge between the HTTP layer and the drift-handling
// logic in resource Read functions (16 resources call isNotFound on the error
// returned by the client).
//
// Covers #118 (404 drift error shape).
func TestServer404SurfacesNotFoundError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "token"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !isNotFound(err) {
		t.Errorf("isNotFound(err) = false; error must match the not-found pattern so resource Read can remove the object from state. err = %v", err)
	}
}
