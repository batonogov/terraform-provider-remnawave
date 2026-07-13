package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		wantErr  string
	}{
		{name: "empty", wantErr: "endpoint is required"},
		{name: "missing scheme", endpoint: "example.com", wantErr: "endpoint scheme must be http or https"},
		{name: "invalid URL", endpoint: "http://[", wantErr: "invalid endpoint"},
		{name: "unsupported scheme", endpoint: "ftp://example.com", wantErr: "endpoint scheme must be http or https"},
		{name: "missing host", endpoint: "http:///api", wantErr: "endpoint must include a host"},
		{name: "query string", endpoint: "https://example.com?debug=true", wantErr: "must not include a query string or fragment"},
		{name: "fragment", endpoint: "https://example.com/#docs", wantErr: "must not include a query string or fragment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewClient(ClientConfig{Endpoint: tt.endpoint})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewClient() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestNewClientConfiguration(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{
		Endpoint:           "https://example.com/remnawave/",
		APIToken:           "api-token",
		Username:           "admin",
		Password:           "secret",
		InsecureSkipVerify: true,
		Timeout:            17 * time.Second,
		ProxyHeaders:       true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if got := client.baseURL.String(); got != "https://example.com/remnawave/" {
		t.Errorf("baseURL = %q", got)
	}
	if client.httpClient.Timeout != 17*time.Second {
		t.Errorf("timeout = %s", client.httpClient.Timeout)
	}
	if client.apiToken != "api-token" || client.username != "admin" || client.password != "secret" {
		t.Errorf("credentials were not retained")
	}
	if !client.proxyHeaders {
		t.Errorf("proxyHeaders = false")
	}

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.httpClient.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify was not configured")
	}

	defaultClient, err := NewClient(ClientConfig{Endpoint: "http://example.com"})
	if err != nil {
		t.Fatalf("NewClient(default) error = %v", err)
	}
	if defaultClient.httpClient.Timeout != 30*time.Second {
		t.Errorf("default timeout = %s, want 30s", defaultClient.httpClient.Timeout)
	}

	defaultTransport := http.DefaultTransport.(*http.Transport)
	if transport == defaultTransport || transport.TLSClientConfig == defaultTransport.TLSClientConfig {
		t.Errorf("NewClient mutated or reused the default transport TLS configuration")
	}
}

func TestResolvePath(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{Endpoint: "https://example.com/panel/"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string
	}{
		{path: "/api/users", want: "https://example.com/panel/api/users"},
		{path: "api/users", want: "https://example.com/panel/api/users"},
		{path: "/api/system/stats?tz=Europe%2FMoscow&label=a+b", want: "https://example.com/panel/api/system/stats?tz=Europe%2FMoscow&label=a+b"},
	}

	for _, tt := range tests {
		if got := client.resolvePath(tt.path); got != tt.want {
			t.Errorf("resolvePath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestStaticTokenRequest(t *testing.T) {
	t.Parallel()

	description := "created by test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/users" {
			t.Errorf("request = %s %s", r.Method, r.URL.RequestURI())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-For"); got != "127.0.0.1" {
			t.Errorf("X-Forwarded-For = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "https" {
			t.Errorf("X-Forwarded-Proto = %q", got)
		}
		if got := r.Header.Get("X-Remnawave-Client-Type"); got != "" {
			t.Errorf("static API token request has X-Remnawave-Client-Type = %q", got)
		}

		var body User
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Username != "alice" || body.ExpireAt != "2027-01-01T00:00:00Z" || body.Description == nil || *body.Description != description {
			t.Errorf("body = %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"uuid":"user-id","username":"alice","expireAt":"2027-01-01T00:00:00Z"}}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token", ProxyHeaders: true})
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateUser(context.Background(), &User{
		Username:    "alice",
		ExpireAt:    "2027-01-01T00:00:00Z",
		Description: &description,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.UUID != "user-id" || created.Username != "alice" {
		t.Errorf("created user = %#v", created)
	}
}

func TestUsernamePasswordAuthenticationIsCached(t *testing.T) {
	t.Parallel()

	var loginCalls atomic.Int32
	var apiCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			loginCalls.Add(1)
			if r.Header.Get("Authorization") != "" {
				t.Errorf("login request unexpectedly has Authorization header")
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode login: %v", err)
			}
			if !reflect.DeepEqual(body, map[string]string{"username": "admin", "password": "secret"}) {
				t.Errorf("login body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"response":{"accessToken":"login-token"}}`)
		case "/api/system/health":
			apiCalls.Add(1)
			if got := r.Header.Get("Authorization"); got != "Bearer login-token" {
				t.Errorf("Authorization = %q", got)
			}
			if got := r.Header.Get("X-Remnawave-Client-Type"); got != "browser" {
				t.Errorf("X-Remnawave-Client-Type = %q", got)
			}
			_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		health, err := client.GetSystemHealth(context.Background())
		if err != nil {
			t.Fatalf("GetSystemHealth() error = %v", err)
		}
		if health["status"] != "ok" {
			t.Errorf("health = %#v", health)
		}
	}
	if loginCalls.Load() != 1 || apiCalls.Load() != 2 {
		t.Errorf("calls: login=%d api=%d", loginCalls.Load(), apiCalls.Load())
	}
}

func TestConcurrentRequestsShareAuthentication(t *testing.T) {
	t.Parallel()

	var loginCalls atomic.Int32
	var apiCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			loginCalls.Add(1)
			time.Sleep(10 * time.Millisecond)
			_, _ = io.WriteString(w, `{"response":{"accessToken":"shared-token"}}`)
		case "/api/system/health":
			apiCalls.Add(1)
			if got := r.Header.Get("Authorization"); got != "Bearer shared-token" {
				t.Errorf("Authorization = %q", got)
			}
			_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	const requests = 20
	start := make(chan struct{})
	errCh := make(chan error, requests)
	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := client.GetSystemHealth(context.Background())
			errCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Errorf("GetSystemHealth() error = %v", err)
		}
	}
	if loginCalls.Load() != 1 {
		t.Errorf("login calls = %d, want 1", loginCalls.Load())
	}
	if apiCalls.Load() != requests {
		t.Errorf("API calls = %d, want %d", apiCalls.Load(), requests)
	}
}

func TestUnauthorizedReauthenticatesAndPreservesBody(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	loginCalls := 0
	apiCalls := 0
	var requestBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/api/auth/login":
			loginCalls++
			_, _ = fmt.Fprintf(w, `{"response":{"accessToken":"token-%d"}}`, loginCalls)
		case "/api/users":
			apiCalls++
			body, _ := io.ReadAll(r.Body)
			requestBodies = append(requestBodies, string(body))
			if r.Header.Get("Authorization") == "Bearer token-1" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"message":"expired"}`)
				return
			}
			_, _ = io.WriteString(w, `{"response":{"uuid":"new-id","username":"alice","expireAt":"2027-01-01T00:00:00Z"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateUser(context.Background(), &User{Username: "alice", ExpireAt: "2027-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.UUID != "new-id" {
		t.Errorf("created = %#v", created)
	}
	if loginCalls != 2 || apiCalls != 2 {
		t.Errorf("calls: login=%d api=%d", loginCalls, apiCalls)
	}
	if len(requestBodies) != 2 || requestBodies[0] == "" || requestBodies[0] != requestBodies[1] {
		t.Errorf("retry bodies = %#v", requestBodies)
	}
}

func TestStaticTokenUnauthorizedIsNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"invalid token"}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "invalid"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemHealth(context.Background())
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestAuthenticationErrors(t *testing.T) {
	t.Parallel()

	t.Run("missing credentials", func(t *testing.T) {
		client, err := NewClient(ClientConfig{Endpoint: "http://example.com"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.token(context.Background())
		if err == nil || !strings.Contains(err.Error(), "either api_token or username+password") {
			t.Fatalf("token() error = %v", err)
		}
	})

	t.Run("empty access token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"response":{}}`)
		}))
		defer server.Close()
		client, err := NewClient(ClientConfig{Endpoint: server.URL, Username: "admin", Password: "secret"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.token(context.Background())
		if err == nil || !strings.Contains(err.Error(), "no access token") {
			t.Fatalf("token() error = %v", err)
		}
	})

	t.Run("login HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"denied"}`)
		}))
		defer server.Close()
		client, err := NewClient(ClientConfig{Endpoint: server.URL, Username: "admin", Password: "secret"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.token(context.Background())
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusForbidden || !strings.Contains(err.Error(), "login failed") {
			t.Fatalf("token() error = %v", err)
		}
	})
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()

	client := &Client{}
	tests := []struct {
		name    string
		status  int
		body    string
		out     any
		want    any
		wantErr string
	}{
		{name: "envelope", status: http.StatusOK, body: `{"response":{"value":"wrapped"}}`, out: &map[string]any{}, want: map[string]any{"value": "wrapped"}},
		{name: "raw object", status: http.StatusOK, body: `{"value":"raw"}`, out: &map[string]any{}, want: map[string]any{"value": "raw"}},
		{name: "raw array", status: http.StatusOK, body: `["a","b"]`, out: &[]string{}, want: []string{"a", "b"}},
		{name: "empty body", status: http.StatusNoContent, out: &map[string]any{}, want: map[string]any{}},
		{name: "nil output", status: http.StatusOK, body: `not-json`},
		{name: "malformed JSON", status: http.StatusOK, body: `{"response":`, out: &map[string]any{}, wantErr: "unexpected end of JSON input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			err := client.decodeResponse(resp, tt.out)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("decodeResponse() error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeResponse() error = %v", err)
			}
			if tt.out != nil {
				got := reflect.ValueOf(tt.out).Elem().Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("decoded = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestDecodeResponseHTTPError(t *testing.T) {
	t.Parallel()

	client := &Client{}
	longBody := strings.Repeat("x", 1100)
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader("  " + longBody + "  ")),
	}
	err := client.decodeResponse(resp, nil)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want *HTTPStatusError", err)
	}
	if statusErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d", statusErr.StatusCode)
	}
	if statusErr.Body != longBody[:1024]+"...(truncated)" {
		t.Errorf("body length/content = %d/%q", len(statusErr.Body), statusErr.Body)
	}
	if got := statusErr.Error(); !strings.Contains(got, "status 502") || !strings.Contains(got, "body:") {
		t.Errorf("Error() = %q", got)
	}
	if got := (&HTTPStatusError{StatusCode: http.StatusNotFound}).Error(); got != "request failed: status 404" {
		t.Errorf("empty-body Error() = %q", got)
	}
}

type failingReadCloser struct {
	err error
}

func (f failingReadCloser) Read([]byte) (int, error) { return 0, f.err }
func (f failingReadCloser) Close() error             { return nil }

func TestDecodeResponseReadError(t *testing.T) {
	t.Parallel()

	want := errors.New("read failed")
	err := (&Client{}).decodeResponse(&http.Response{
		StatusCode: http.StatusOK,
		Body:       failingReadCloser{err: want},
	}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRequestErrors(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{Endpoint: "http://example.com", APIToken: "token"})
	if err != nil {
		t.Fatal(err)
	}
	err = client.doRequest(context.Background(), http.MethodPost, "/api/test", make(chan int), nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("marshal error = %v", err)
	}

	want := errors.New("transport failed")
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	})
	_, err = client.GetSystemHealth(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("transport error = %v, want %v", err, want)
	}
}

func TestSetProxyHeadersDisabled(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	(&Client{}).setProxyHeaders(req)
	if req.Header.Get("X-Forwarded-For") != "" || req.Header.Get("X-Forwarded-Proto") != "" {
		t.Errorf("proxy headers unexpectedly set: %#v", req.Header)
	}
}
