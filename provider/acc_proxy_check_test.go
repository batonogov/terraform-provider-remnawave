package provider

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestAccProductionProxyCheck verifies the real middleware, not the provider's
// header helpers. CI runs the entire acceptance matrix in production mode.
func TestAccProductionProxyCheck(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("REMNAWAVE_NODE_ENV") != "production" {
		t.Skip("requires a panel started with REMNAWAVE_NODE_ENV=production")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	endpoint := strings.TrimRight(os.Getenv(envEndpoint), "/")

	checkAllowed := func(t *testing.T, method, path, body string, wantStatus int) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, endpoint+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.Header.Set("X-Forwarded-Proto", "https")
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request with valid proxy headers: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != wantStatus {
			t.Fatalf("status = %d, want %d", resp.StatusCode, wantStatus)
		}
	}

	t.Run("valid_proxy_headers", func(t *testing.T) {
		checkAllowed(t, http.MethodGet, "/api/auth/status", "", http.StatusOK)
	})

	type rejectedRequest struct {
		name   string
		method string
		path   string
		body   string
		forIP  string
		proto  string
	}
	cases := []rejectedRequest{
		{"missing_both_headers", http.MethodGet, "/api/auth/status", "", "", ""},
		{"missing_forwarded_for", http.MethodGet, "/api/auth/status", "", "", "https"},
		{"missing_forwarded_proto", http.MethodGet, "/api/auth/status", "", "127.0.0.1", ""},
		{"insecure_forwarded_proto", http.MethodGet, "/api/auth/status", "", "127.0.0.1", "http"},
	}

	// 3.4.11 moves proxy validation before JSON parsing. With valid proxy
	// headers malformed JSON reaches the parser (400); without them the socket
	// must be destroyed before parsing rather than returning a JSON error.
	version := os.Getenv("REMNAWAVE_VERSION")
	major, minor, patch, ok := parseVersion(version)
	if version == "" || ok && (major > 3 || major == 3 && (minor > 4 || minor == 4 && patch >= 11)) {
		t.Run("malformed_json_with_proxy_headers", func(t *testing.T) {
			checkAllowed(t, http.MethodPost, "/api/auth/login", "{", http.StatusBadRequest)
		})
		cases = append(cases, rejectedRequest{
			"proxy_check_precedes_json_parser", http.MethodPost, "/api/auth/login", "{", "", "",
		})
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tt.method, endpoint+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			if tt.forIP != "" {
				req.Header.Set("X-Forwarded-For", tt.forIP)
			}
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := client.Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if err == nil {
				t.Fatalf("request bypassed proxy check: HTTP %d", resp.StatusCode)
			}
			if !errors.Is(err, io.EOF) && !errors.Is(err, syscall.ECONNRESET) {
				t.Fatalf("expected connection closed by proxy middleware, got %v", err)
			}
		})
	}

	t.Run("panel_remains_available", func(t *testing.T) {
		checkAllowed(t, http.MethodGet, "/api/auth/status", "", http.StatusOK)
	})
}
