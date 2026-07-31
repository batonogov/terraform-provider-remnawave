package provider

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTransportProtocolErrorsAreOpaque(t *testing.T) {
	const (
		apiToken      = "reflected-api-token"
		loginPassword = "reflected-login-password"
		customHeader  = "reflected-gateway-secret"
	)

	tests := []struct {
		name       string
		config     ClientConfig
		response   func(*http.Request, []byte, int) string
		forbidden  []string
		wantPrefix string
	}{
		{
			name: "bearer token in malformed status",
			config: ClientConfig{
				APIToken: apiToken,
			},
			response: func(req *http.Request, _ []byte, _ int) string {
				reflected := req.Header.Get("Authorization")
				if reflected != "Bearer "+apiToken {
					return "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
				}
				return "HTTP/1.1 " + reflected + "\r\n\r\n"
			},
			forbidden: []string{apiToken, "Bearer " + apiToken},
		},
		{
			name: "transformed login password in malformed status",
			config: ClientConfig{
				Username: "admin",
				Password: loginPassword,
			},
			response: func(_ *http.Request, body []byte, _ int) string {
				if !strings.Contains(string(body), loginPassword) {
					return "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
				}
				transformed := base64.RawURLEncoding.EncodeToString([]byte(loginPassword))
				return "HTTP/1.1 " + transformed + "\r\n\r\n"
			},
			forbidden: []string{
				loginPassword,
				base64.RawURLEncoding.EncodeToString([]byte(loginPassword)),
			},
			wantPrefix: "login failed: ",
		},
		{
			name: "custom header in malformed response header",
			config: ClientConfig{
				APIToken:      "static-token",
				CustomHeaders: map[string]string{"X-Gateway-Token": customHeader},
			},
			response: func(req *http.Request, _ []byte, _ int) string {
				reflected := req.Header.Get("X-Gateway-Token")
				if reflected != customHeader {
					return "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
				}
				return "HTTP/1.1 200 OK\r\nX-Reflected: " + reflected + "\x00\r\n\r\n"
			},
			forbidden: []string{customHeader},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := startRawHTTPServer(t, 1, tt.response)
			tt.config.Endpoint = endpoint
			client, err := NewClient(tt.config)
			if err != nil {
				t.Fatal(err)
			}

			_, err = client.GetSystemHealth(context.Background())
			if err == nil {
				t.Fatal("GetSystemHealth() error = nil, want transport error")
			}
			if !errors.Is(err, errHTTPRequestFailed) {
				t.Fatalf("GetSystemHealth() error = %v, want opaque request error", err)
			}
			if tt.wantPrefix != "" && !strings.HasPrefix(err.Error(), tt.wantPrefix) {
				t.Errorf("error = %q, want prefix %q", err, tt.wantPrefix)
			}
			assertOpaqueRequestError(t, err, tt.forbidden...)
		})
	}
}

func TestSameOriginRedirectTransportErrorIsOpaque(t *testing.T) {
	const secret = "redirect-reflected-secret/value"
	transformed := base64.RawURLEncoding.EncodeToString([]byte(secret))
	endpoint := startRawHTTPServer(t, 2, func(_ *http.Request, _ []byte, requestNumber int) string {
		if requestNumber == 0 {
			return "HTTP/1.1 302 Found\r\n" +
				"Location: /redirect/" + url.PathEscape(secret) + "\r\n" +
				"Content-Length: 0\r\n" +
				"Connection: close\r\n\r\n"
		}
		return "HTTP/1.1 " + transformed + "\r\n\r\n"
	})

	client, err := NewClient(ClientConfig{Endpoint: endpoint, APIToken: "static-token"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemHealth(context.Background())
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatalf("GetSystemHealth() error = %v, want opaque request error", err)
	}
	assertOpaqueRequestError(t, err, secret, url.PathEscape(secret), transformed)
}

func TestSensitiveRequestURLIsOmittedFromTransportError(t *testing.T) {
	const shortUUID = "sensitive-short-uuid"
	secretBearingCause := errors.New("transport inspected " + shortUUID)
	client, err := NewClient(ClientConfig{
		Endpoint: "http://example.test",
		APIToken: "static-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, secretBearingCause
	})

	_, err = client.GetSubscriptionByShortUUID(context.Background(), shortUUID)
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatalf("GetSubscriptionByShortUUID() error = %v, want opaque request error", err)
	}
	if errors.Is(err, secretBearingCause) {
		t.Fatal("request error retained the secret-bearing transport cause")
	}
	assertOpaqueRequestError(t, err, shortUUID)
}

func TestSafeRequestErrorCategoriesRemainDetectable(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ClientConfig{
				Endpoint: "http://example.test",
				APIToken: "static-token",
			})
			if err != nil {
				t.Fatal(err)
			}
			client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, tt.err
			})

			_, err = client.GetSystemHealth(context.Background())
			if !errors.Is(err, tt.err) {
				t.Fatalf("GetSystemHealth() error = %v, want %v", err, tt.err)
			}
			var urlErr *url.Error
			if errors.As(err, &urlErr) {
				t.Fatalf("safe request error retained *url.Error: %v", urlErr)
			}
		})
	}
}

func assertOpaqueRequestError(t *testing.T, err error, forbidden ...string) {
	t.Helper()

	for _, value := range forbidden {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Errorf("request error disclosed %q: %v", value, err)
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		t.Errorf("request error retained unsafe *url.Error: %v", urlErr)
	}
}

func startRawHTTPServer(
	t *testing.T,
	requestCount int,
	response func(*http.Request, []byte, int) string,
) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		for requestNumber := 0; requestNumber < requestCount; requestNumber++ {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				result <- acceptErr
				return
			}
			if serveErr := serveRawHTTPConnection(conn, requestNumber, response); serveErr != nil {
				result <- serveErr
				return
			}
		}
		result <- nil
	}()

	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case serveErr := <-result:
			if serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
				t.Errorf("raw HTTP server: %v", serveErr)
			}
		case <-time.After(5 * time.Second):
			t.Error("raw HTTP server did not stop")
		}
	})

	return "http://" + listener.Addr().String()
}

func serveRawHTTPConnection(
	conn net.Conn,
	requestNumber int,
	response func(*http.Request, []byte, int) string,
) error {
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	if _, err := io.WriteString(conn, response(req, body, requestNumber)); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

// TestConnectionRefusedTransportErrorIsOpaque verifies that a connection
// refused (port freed by a closed server) collapses to the opaque
// errHTTPRequestFailed sentinel without leaking the host:port or URL.
func TestConnectionRefusedTransportErrorIsOpaque(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := server.URL
	server.Close() // free the port so the next dial is refused

	client, err := NewClient(ClientConfig{Endpoint: addr, APIToken: "refused-secret"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("GetSystemHealth() error = nil, want connection-refused error")
	}
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatalf("GetSystemHealth() error = %v, want opaque errHTTPRequestFailed", err)
	}
	// The closed server's host:port and full URL must not appear in the error.
	host := addr[len("http://"):]
	if strings.Contains(err.Error(), host) {
		t.Errorf("error disclosed closed-server host:port %q: %v", host, err)
	}
	if strings.Contains(err.Error(), addr) {
		t.Errorf("error disclosed closed-server URL %q: %v", addr, err)
	}
}

// TestClient_5xx_NoRetry_Intentional documents and locks the intentional
// design: the client performs NO retry or backoff on 5xx server errors. A
// single failing attempt surfaces as HTTPStatusError{StatusCode:500}.
func TestClient_5xx_NoRetry_Intentional(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("GetSystemHealth() error = nil, want 500 error")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("error = %v, want HTTPStatusError{StatusCode:500}", err)
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("server attempts = %d, want 1 (no retry on 5xx)", got)
	}
}

// TestClient_Timeout_SlowBackend_PropagatesAsDeadlineExceeded verifies that a
// request which exceeds the client's configured timeout surfaces as a context
// deadline error (or an opaque error that does not leak the URL).
func TestClient_Timeout_SlowBackend_PropagatesAsDeadlineExceeded(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block well past the client timeout.
		time.Sleep(300 * time.Millisecond)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Endpoint: server.URL,
		APIToken: "timeout-secret",
		Timeout:  50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetSystemHealth(context.Background())
	if err == nil {
		t.Fatal("GetSystemHealth() error = nil, want timeout error")
	}
	// The http.Client timeout wraps the dial/read/write deadlines as a
	// context.DeadlineExceeded, which sanitizeRequestError passes through.
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
	// The endpoint URL and secret must never leak via the timeout error.
	if strings.Contains(err.Error(), server.URL) {
		t.Errorf("timeout error disclosed endpoint URL: %v", err)
	}
}
