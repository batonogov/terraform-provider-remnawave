package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCustomHeadersValidation(t *testing.T) {
	t.Parallel()

	t.Run("allowed and canonicalized", func(t *testing.T) {
		t.Parallel()
		client, err := NewClient(ClientConfig{
			Endpoint: "https://example.com",
			CustomHeaders: map[string]string{
				"cOoKiE":              "\tgate=value ",
				"cf-access-client-id": "client-id",
			},
		})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if got := client.customHeaders["Cookie"]; got != "gate=value" {
			t.Errorf("Cookie = %q, want canonicalized value", got)
		}
		if got := client.customHeaders["Cf-Access-Client-Id"]; got != "client-id" {
			t.Errorf("Cf-Access-Client-Id = %q", got)
		}
	})

	t.Run("case-insensitive duplicate", func(t *testing.T) {
		t.Parallel()
		secret := "must-not-appear"
		_, err := NewClient(ClientConfig{
			Endpoint: "https://example.com",
			CustomHeaders: map[string]string{
				"Cookie": secret,
				"cookie": secret,
			},
		})
		if err == nil || !strings.Contains(err.Error(), "duplicate custom header") {
			t.Fatalf("NewClient() error = %v, want duplicate-header error", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error disclosed header value: %v", err)
		}
	})

	reserved := []string{
		"Authorization",
		"Proxy-Authorization",
		"Proxy-Authenticate",
		"Host",
		"Content-Length",
		"Content-Type",
		"Transfer-Encoding",
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"TE",
		"Trailer",
		"Upgrade",
		"HTTP2-Settings",
		"X-Remnawave-Client-Type",
		"X-Forwarded-For",
		"x-forwarded-custom",
	}
	for _, name := range reserved {
		name := name
		t.Run("reserved "+name, func(t *testing.T) {
			t.Parallel()
			secret := "reserved-secret"
			_, err := NewClient(ClientConfig{
				Endpoint:      "https://example.com",
				CustomHeaders: map[string]string{name: secret},
			})
			if err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("NewClient() error = %v, want reserved-header error", err)
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error disclosed header value: %v", err)
			}
		})
	}

	invalidNames := []string{"", "Bad Header", "Bad:Header", "Bad\r\nInjected", "X-Unicodé"}
	for i, name := range invalidNames {
		name := name
		t.Run(fmt.Sprintf("invalid name %d", i), func(t *testing.T) {
			t.Parallel()
			secret := "invalid-name-secret"
			_, err := NewClient(ClientConfig{
				Endpoint:      "https://example.com",
				CustomHeaders: map[string]string{name: secret},
			})
			if err == nil || !strings.Contains(err.Error(), "invalid custom header name") {
				t.Fatalf("NewClient() error = %v, want invalid-name error", err)
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error disclosed header value: %v", err)
			}
		})
	}

	invalidValues := []string{
		"do-not-disclose\r\nInjected: true",
		"do-not-disclose\n",
		"do-not-disclose\x00",
		"do-not-disclose\x7f",
	}
	for i, value := range invalidValues {
		value := value
		t.Run(fmt.Sprintf("invalid value %d", i), func(t *testing.T) {
			t.Parallel()
			_, err := NewClient(ClientConfig{
				Endpoint:      "https://example.com",
				CustomHeaders: map[string]string{"X-Gateway-Secret": value},
			})
			if err == nil || !strings.Contains(err.Error(), "invalid value") {
				t.Fatalf("NewClient() error = %v, want invalid-value error", err)
			}
			if strings.Contains(err.Error(), "do-not-disclose") {
				t.Fatalf("error disclosed header value: %v", err)
			}
		})
	}
}

func TestCustomHeadersStaticTokenAndGate(t *testing.T) {
	t.Parallel()

	const cookie = "gateway=secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != cookie {
			w.WriteHeader(444)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Cf-Access-Client-Secret"); got != "cf-secret" {
			t.Errorf("Cf-Access-Client-Secret = %q", got)
		}
		_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
	}))
	t.Cleanup(server.Close)

	withoutGateHeader, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = withoutGateHeader.GetSystemHealth(context.Background())
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != 444 {
		t.Fatalf("request without cookie error = %v, want HTTP 444", err)
	}

	withGateHeader, err := NewClient(ClientConfig{
		Endpoint: server.URL,
		APIToken: "static-token",
		CustomHeaders: map[string]string{
			"Cookie":                  cookie,
			"CF-Access-Client-Secret": "cf-secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	health, err := withGateHeader.GetSystemHealth(context.Background())
	if err != nil {
		t.Fatalf("GetSystemHealth() error = %v", err)
	}
	if health["status"] != "ok" {
		t.Errorf("health = %#v", health)
	}
}

func TestCustomHeadersLoginAndUnauthorizedRetry(t *testing.T) {
	t.Parallel()

	const cookie = "gateway=secret"
	var mu sync.Mutex
	loginCalls := 0
	apiCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if got := r.Header.Get("Cookie"); got != cookie {
			t.Errorf("%s Cookie = %q", r.URL.Path, got)
		}

		switch r.URL.Path {
		case "/api/auth/login":
			loginCalls++
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("login Authorization = %q", got)
			}
			_, _ = fmt.Fprintf(w, `{"response":{"accessToken":"token-%d"}}`, loginCalls)
		case "/api/system/health":
			apiCalls++
			if got := r.Header.Get("X-Remnawave-Client-Type"); got != "browser" {
				t.Errorf("X-Remnawave-Client-Type = %q", got)
			}
			if r.Header.Get("Authorization") == "Bearer token-1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-2" {
				t.Errorf("retry Authorization = %q", got)
			}
			_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		Endpoint:      server.URL,
		Username:      "admin",
		Password:      "secret",
		CustomHeaders: map[string]string{"Cookie": cookie},
	})
	if err != nil {
		t.Fatal(err)
	}
	health, err := client.GetSystemHealth(context.Background())
	if err != nil {
		t.Fatalf("GetSystemHealth() error = %v", err)
	}
	if health["status"] != "ok" {
		t.Errorf("health = %#v", health)
	}
	if loginCalls != 2 || apiCalls != 2 {
		t.Errorf("calls: login=%d api=%d, want 2 each", loginCalls, apiCalls)
	}
}

func TestCustomHeadersVersionDetection(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/system/metadata" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Cookie"); got != "gateway=version-secret" {
			t.Errorf("Cookie = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = io.WriteString(w, `{"response":{"version":"2.8.1"}}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		Endpoint:      server.URL,
		APIToken:      "static-token",
		CustomHeaders: map[string]string{"Cookie": "gateway=version-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.detectVersion(context.Background()); err != nil {
		t.Fatalf("detectVersion() error = %v", err)
	}
	if err := client.detectVersion(context.Background()); err != nil {
		t.Fatalf("cached detectVersion() error = %v", err)
	}
	if client.serverVersion != "2.8" || calls.Load() != 1 {
		t.Errorf("serverVersion/calls = %q/%d, want 2.8/1", client.serverVersion, calls.Load())
	}
}

func TestCustomHeaderHTTPErrorBodyIsOmitted(t *testing.T) {
	t.Parallel()

	const secret = "gateway=a b/c"
	tests := []struct {
		name string
		body string
	}{
		{
			name: "arbitrary proxy details",
			body: `{"message":"upstream gateway rejected the request","request_id":"public-id"}`,
		},
		{
			name: "URL-encoded complete header value",
			body: "gateway%3Da+b%2Fc",
		},
		{
			name: "cookie value without name",
			body: "a b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)

			client, err := NewClient(ClientConfig{
				Endpoint:      server.URL,
				APIToken:      "static-token",
				CustomHeaders: map[string]string{"Cookie": secret},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetSystemHealth(context.Background())
			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("error = %v, want *HTTPStatusError", err)
			}
			if statusErr.StatusCode != http.StatusBadGateway {
				t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusBadGateway)
			}
			if statusErr.Body != "" {
				t.Errorf("HTTPStatusError.Body = %q, want empty", statusErr.Body)
			}
			if got := statusErr.Error(); got != "request failed: status 502" {
				t.Errorf("HTTPStatusError.Error() = %q", got)
			}
			if strings.Contains(err.Error(), tt.body) {
				t.Fatalf("error disclosed untrusted response body: %v", err)
			}
		})
	}
}

func TestCustomHeaderResponseBodyReadErrorIsOpaque(t *testing.T) {
	t.Parallel()

	const secret = "gateway=a b/c"
	readErr := errors.New("body read failed: gateway%3Da+b%2Fc")
	client, err := NewClient(ClientConfig{
		Endpoint:      "http://example.test",
		APIToken:      "static-token",
		CustomHeaders: map[string]string{"Cookie": secret},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       failingReadCloser{err: readErr},
			Request:    req,
		}, nil
	})

	_, err = client.GetSystemHealth(context.Background())
	if !errors.Is(err, errCustomHeaderResponseBodyRead) {
		t.Fatalf("error = %v, want fixed response-body read error", err)
	}
	if errors.Is(err, readErr) {
		t.Fatal("response-body read error retained the secret-bearing cause")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "gateway%3Da+b%2Fc") {
		t.Fatalf("response-body read error disclosed a header value: %v", err)
	}
}

func TestCustomHeaderHTTPErrorBodyIsNotRead(t *testing.T) {
	t.Parallel()

	const encodedSecret = "gateway%3Da+b%2Fc"
	readErr := errors.New("body read failed: " + encodedSecret)
	client, err := NewClient(ClientConfig{
		Endpoint:      "http://example.test",
		APIToken:      "static-token",
		CustomHeaders: map[string]string{"Cookie": "gateway=a b/c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     make(http.Header),
			Body:       failingReadCloser{err: readErr},
			Request:    req,
		}, nil
	})

	_, err = client.GetSystemHealth(context.Background())
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadGateway || statusErr.Body != "" {
		t.Fatalf("error = %#v, want status-only HTTP 502 error", err)
	}
	if errors.Is(err, readErr) || strings.Contains(err.Error(), encodedSecret) {
		t.Fatalf("HTTP error attempted to expose its response body: %v", err)
	}
}

func TestCustomHeaderValuesAreRedactedFromRequestErrors(t *testing.T) {
	t.Parallel()

	const secret = "gateway=transport-secret"
	transportErr := errors.New("transport reflected " + secret)
	client, err := NewClient(ClientConfig{
		Endpoint:      "http://example.test",
		APIToken:      "static-token",
		CustomHeaders: map[string]string{"Cookie": secret},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, transportErr
	})

	_, err = client.GetSystemHealth(context.Background())
	if err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("GetSystemHealth() error = %v, want redacted transport error", err)
	}
	if errors.Is(err, transportErr) {
		t.Errorf("redacted request error exposed the secret-bearing transport error")
	}
}

func TestCustomHeadersDefensiveCopyConcurrentRequests(t *testing.T) {
	t.Parallel()

	configured := map[string]string{"Cookie": "gateway=original"}
	client, err := NewClient(ClientConfig{
		Endpoint:      "http://example.test",
		APIToken:      "static-token",
		CustomHeaders: configured,
	})
	if err != nil {
		t.Fatal(err)
	}
	configured["Cookie"] = "gateway=mutated"

	var badHeader atomic.Bool
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Cookie") != "gateway=original" {
			badHeader.Store(true)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"response":{"status":"ok"}}`)),
			Request:    req,
		}, nil
	})

	stopWriter := make(chan struct{})
	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		for i := 0; ; i++ {
			select {
			case <-stopWriter:
				return
			default:
				configured["Cookie"] = fmt.Sprintf("gateway=concurrent-mutation-%d", i)
			}
		}
	}()

	const requests = 50
	var requestWG sync.WaitGroup
	errCh := make(chan error, requests)
	for range requests {
		requestWG.Add(1)
		go func() {
			defer requestWG.Done()
			_, err := client.GetSystemHealth(context.Background())
			errCh <- err
		}()
	}
	requestWG.Wait()
	close(errCh)
	close(stopWriter)
	writerWG.Wait()

	for err := range errCh {
		if err != nil {
			t.Errorf("GetSystemHealth() error = %v", err)
		}
	}
	if badHeader.Load() {
		t.Error("client observed a mutation of the input custom-headers map")
	}
}

func TestCustomHeaderRedirects(t *testing.T) {
	t.Parallel()

	t.Run("same origin", func(t *testing.T) {
		var targetCalls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/system/health":
				http.Redirect(w, r, "/redirected", http.StatusFound)
			case "/redirected":
				targetCalls.Add(1)
				if got := r.Header.Get("Cookie"); got != "gateway=same-origin" {
					t.Errorf("redirected Cookie = %q", got)
				}
				_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{
			Endpoint:      server.URL,
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": "gateway=same-origin"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.GetSystemHealth(context.Background()); err != nil {
			t.Fatalf("GetSystemHealth() error = %v", err)
		}
		if targetCalls.Load() != 1 {
			t.Errorf("redirect target calls = %d, want 1", targetCalls.Load())
		}
	})

	t.Run("implicit and explicit default ports", func(t *testing.T) {
		tests := []struct {
			endpoint string
			location string
		}{
			{endpoint: "http://example.test", location: "http://EXAMPLE.test:80/final"},
			{endpoint: "https://example.test", location: "https://EXAMPLE.test:443/final"},
		}
		for _, tt := range tests {
			t.Run(tt.endpoint, func(t *testing.T) {
				client, err := NewClient(ClientConfig{
					Endpoint:      tt.endpoint,
					APIToken:      "static-token",
					CustomHeaders: map[string]string{"Cookie": "gateway=default-port"},
				})
				if err != nil {
					t.Fatal(err)
				}
				var calls atomic.Int32
				client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
					call := calls.Add(1)
					if call == 1 {
						return redirectResponse(req, tt.location), nil
					}
					if req.URL.String() != tt.location {
						t.Errorf("redirect URL = %q, want %q", req.URL, tt.location)
					}
					if got := req.Header.Get("Cookie"); got != "gateway=default-port" {
						t.Errorf("redirected Cookie = %q", got)
					}
					if got := req.Header.Get("Authorization"); got != "Bearer static-token" {
						t.Errorf("redirected Authorization = %q", got)
					}
					return jsonResponse(req, http.StatusOK, `{"response":{"status":"ok"}}`), nil
				})
				if _, err := client.GetSystemHealth(context.Background()); err != nil {
					t.Fatalf("GetSystemHealth() error = %v", err)
				}
				if calls.Load() != 2 {
					t.Errorf("round trips = %d, want 2", calls.Load())
				}
			})
		}
	})

	t.Run("different hostname", func(t *testing.T) {
		var targetCalls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/target" {
				targetCalls.Add(1)
				return
			}
			serverURL, _ := url.Parse("http://localhost")
			currentURL, _ := url.Parse(serverURLForRequest(r))
			serverURL.Host = "localhost:" + currentURL.Port()
			serverURL.Path = "/target"
			http.Redirect(w, r, serverURL.String(), http.StatusFound)
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{
			Endpoint:      server.URL,
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": "gateway=hostname"},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.GetSystemHealth(context.Background())
		if err == nil || !strings.Contains(err.Error(), "different origin") {
			t.Fatalf("GetSystemHealth() error = %v, want cross-origin redirect error", err)
		}
		if targetCalls.Load() != 0 {
			t.Errorf("cross-origin target calls = %d, want 0", targetCalls.Load())
		}
	})

	t.Run("different effective port", func(t *testing.T) {
		var targetCalls atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			targetCalls.Add(1)
		}))
		defer target.Close()
		source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, http.StatusFound)
		}))
		defer source.Close()

		client, err := NewClient(ClientConfig{
			Endpoint:      source.URL,
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": "gateway=port"},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.GetSystemHealth(context.Background())
		if err == nil || !strings.Contains(err.Error(), "different origin") {
			t.Fatalf("GetSystemHealth() error = %v, want cross-origin redirect error", err)
		}
		if targetCalls.Load() != 0 {
			t.Errorf("cross-origin target calls = %d, want 0", targetCalls.Load())
		}
	})

	t.Run("HTTPS to HTTP", func(t *testing.T) {
		var roundTrips atomic.Int32
		client, err := NewClient(ClientConfig{
			Endpoint:      "https://example.test:443",
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": "gateway=downgrade"},
		})
		if err != nil {
			t.Fatal(err)
		}
		client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			roundTrips.Add(1)
			return redirectResponse(req, "http://example.test:443/target"), nil
		})
		_, err = client.GetSystemHealth(context.Background())
		if err == nil || !strings.Contains(err.Error(), "different origin") {
			t.Fatalf("GetSystemHealth() error = %v, want cross-origin redirect error", err)
		}
		if roundTrips.Load() != 1 {
			t.Errorf("round trips = %d, want 1 (redirect target must not be requested)", roundTrips.Load())
		}
	})

	t.Run("same-origin redirect error omits reflected URL", func(t *testing.T) {
		const secret = "gateway=same origin/path"
		escapedSecret := url.PathEscape(secret)
		client, err := NewClient(ClientConfig{
			Endpoint:      "http://example.test",
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": secret},
		})
		if err != nil {
			t.Fatal(err)
		}
		var roundTrips atomic.Int32
		client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if roundTrips.Add(1) == 1 {
				return redirectResponse(req, "http://example.test/"+escapedSecret), nil
			}
			return nil, errors.New("connection closed")
		})

		_, err = client.GetSystemHealth(context.Background())
		if err == nil {
			t.Fatal("GetSystemHealth() error = nil, want transport error")
		}
		if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), escapedSecret) {
			t.Fatalf("same-origin redirect error disclosed reflected header value: %v", err)
		}
		if !strings.Contains(err.Error(), "connection closed") {
			t.Errorf("sanitized error lost safe transport detail: %v", err)
		}
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			t.Errorf("sanitized error exposed secret-bearing *url.Error: %v", urlErr)
		}
		if roundTrips.Load() != 2 {
			t.Errorf("round trips = %d, want 2", roundTrips.Load())
		}
	})

	t.Run("reflected secret in Location is redacted", func(t *testing.T) {
		const secret = "gateway=a b/c"
		escapedSecret := url.PathEscape(secret)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://different.example/"+escapedSecret)
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{
			Endpoint:      server.URL,
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": secret},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.GetSystemHealth(context.Background())
		if err == nil {
			t.Fatal("GetSystemHealth() error = nil, want redirect error")
		}
		if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), escapedSecret) {
			t.Fatalf("redirect error disclosed configured header value: %v", err)
		}
		if !errors.Is(err, errCrossOriginRedirect) {
			t.Errorf("redirect error = %v, want static cross-origin policy error", err)
		}
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			t.Errorf("redacted error exposed secret-bearing *url.Error: %v", urlErr)
		}
	})

	t.Run("malformed reflected Location is collapsed", func(t *testing.T) {
		const secret = "gateway=malformed a/b"
		escapedSecret := url.PathEscape(secret)
		client, err := NewClient(ClientConfig{
			Endpoint:      "http://example.test",
			APIToken:      "static-token",
			CustomHeaders: map[string]string{"Cookie": secret},
		})
		if err != nil {
			t.Fatal(err)
		}
		var roundTrips atomic.Int32
		client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			roundTrips.Add(1)
			return redirectResponse(req, "http://different.example/%zz/"+escapedSecret), nil
		})

		_, err = client.GetSystemHealth(context.Background())
		if !errors.Is(err, errInvalidRedirectLocation) {
			t.Fatalf("GetSystemHealth() error = %v, want static invalid-Location error", err)
		}
		if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), escapedSecret) {
			t.Fatalf("malformed redirect error disclosed configured header value: %v", err)
		}
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			t.Errorf("invalid-Location error exposed secret-bearing *url.Error: %v", urlErr)
		}
		if roundTrips.Load() != 1 {
			t.Errorf("round trips = %d, want 1", roundTrips.Load())
		}
	})

	t.Run("cross-origin redirects rejected when unset", func(t *testing.T) {
		var targetCalls atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			targetCalls.Add(1)
			_, _ = io.WriteString(w, `{"response":{"status":"ok"}}`)
		}))
		defer target.Close()
		source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, http.StatusFound)
		}))
		defer source.Close()

		client, err := NewClient(ClientConfig{Endpoint: source.URL, APIToken: "static-token"})
		if err != nil {
			t.Fatal(err)
		}
		if client.httpClient.CheckRedirect == nil {
			t.Fatal("CheckRedirect is not set without custom headers")
		}
		if _, err := client.GetSystemHealth(context.Background()); !errors.Is(err, errCrossOriginRedirect) {
			t.Fatalf("GetSystemHealth() error = %v, want cross-origin redirect error", err)
		}
		if targetCalls.Load() != 0 {
			t.Errorf("redirect target calls = %d, want 0", targetCalls.Load())
		}
	})
}

func redirectResponse(req *http.Request, location string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{location}},
		Body:       io.NopCloser(strings.NewReader("redirect")),
		Request:    req,
	}
}

func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func serverURLForRequest(r *http.Request) string {
	return "http://" + r.Host
}
