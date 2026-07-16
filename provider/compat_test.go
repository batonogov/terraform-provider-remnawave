package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestParseMajorMinor validates the semver major.minor extraction.
func TestParseMajorMinor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"2.7.4", "2.7"},
		{"2.8.0", "2.8"},
		{"2.7", "2.7"},
		{"3.0.1", "3.0"},
		{"1", ""},
		{"", ""},
		{"garbage", ""},
		{"v2.8.0", "v2.8"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := parseMajorMinor(tt.input); got != tt.want {
				t.Errorf("parseMajorMinor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestVersionDetection verifies that the client lazily detects and caches
// the server version from /api/system/metadata.
func TestVersionDetection(t *testing.T) {
	t.Parallel()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	// Before detection: version is empty
	if got := client.serverVersion; got != "" {
		t.Errorf("serverVersion before detection = %q, want empty", got)
	}

	// Trigger detection
	if !client.isVersion2_7(context.Background()) {
		t.Error("isVersion2_7() = false, want true for 2.7.4")
	}

	if got := client.serverVersion; got != "2.7" {
		t.Errorf("serverVersion = %q, want 2.7", got)
	}

	// Detection should only call /api/system/metadata once
	client.isVersion2_7(context.Background())
	client.isVersion2_7(context.Background())
	if requestCount != 1 {
		t.Errorf("metadata endpoint called %d times, want 1 (cached)", requestCount)
	}
}

// TestVersionDetection2_8 verifies version detection for 2.8.x.
func TestVersionDetection2_8(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.8.0"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	if client.isVersion2_7(context.Background()) {
		t.Error("isVersion2_7() = true, want false for 2.8.0")
	}
}

// TestVersionDetectionFailure verifies that a failed metadata request
// does not crash the client — version stays empty and isVersion2_7 returns false.
func TestVersionDetectionFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	// Should not panic — defaults to false (2.8 behaviour)
	if client.isVersion2_7(context.Background()) {
		t.Error("isVersion2_7() = true after detection failure, want false")
	}
}

// TestCreateApiTokenV27 verifies the 2.7.x token creation path: the
// request uses tokenName (not name), and the response is adapted.
func TestCreateApiTokenV27(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		if r.URL.Path != "/api/tokens" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s, want POST /api/tokens", r.Method, r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request JSON: %v", err)
		}

		// 2.7.x should send tokenName, not name/expiresInDays/scopes
		if v, ok := got["tokenName"]; !ok || v != "my-token" {
			t.Errorf("request tokenName = %v, want \"my-token\"", v)
		}
		if _, ok := got["name"]; ok {
			t.Error("request should not contain \"name\" field for 2.7.x")
		}
		if _, ok := got["expiresInDays"]; ok {
			t.Error("request should not contain \"expiresInDays\" field for 2.7.x")
		}
		if _, ok := got["scopes"]; ok {
			t.Error("request should not contain \"scopes\" field for 2.7.x")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"uuid":"tok-uuid","token":"jwt-value","tokenName":"my-token"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	token, err := client.CreateApiToken(context.Background(), &ApiToken{
		Name:          "my-token",
		ExpiresInDays: 7,
		Scopes:        []string{"*"},
	})
	if err != nil {
		t.Fatalf("CreateApiToken() error = %v", err)
	}

	if token.UUID != "tok-uuid" {
		t.Errorf("UUID = %q, want \"tok-uuid\"", token.UUID)
	}
	if token.Name != "my-token" {
		t.Errorf("Name = %q, want \"my-token\"", token.Name)
	}
	if token.Token != "jwt-value" {
		t.Errorf("Token = %q, want \"jwt-value\"", token.Token)
	}
}

// TestGetAllApiTokensV27 verifies the 2.7.x token list path: the
// response uses apiKeys[] with tokenName, not tokens[] with name.
func TestGetAllApiTokensV27(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		if r.URL.Path != "/api/tokens" || r.Method != http.MethodGet {
			t.Errorf("request = %s %s, want GET /api/tokens", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"apiKeys":[{"uuid":"uuid-1","tokenName":"token-1","token":"redacted","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},{"uuid":"uuid-2","tokenName":"token-2","token":"redacted","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"docs":{"isDocsEnabled":true,"scalarPath":"/scalar","swaggerPath":"/docs"}}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := client.GetAllApiTokens(context.Background())
	if err != nil {
		t.Fatalf("GetAllApiTokens() error = %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("len(tokens) = %d, want 2", len(tokens))
	}
	if tokens[0].UUID != "uuid-1" || tokens[0].Name != "token-1" {
		t.Errorf("tokens[0] = %+v", tokens[0])
	}
	if tokens[1].UUID != "uuid-2" || tokens[1].Name != "token-2" {
		t.Errorf("tokens[1] = %+v", tokens[1])
	}
}

// TestCreateApiTokenV28 verifies the 2.8.x token creation path is unchanged:
// request uses name, expiresInDays, scopes.
func TestCreateApiTokenV28(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.8.0"}}`)
			return
		}
		if r.URL.Path != "/api/tokens" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s, want POST /api/tokens", r.Method, r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request JSON: %v", err)
		}

		// 2.8.x should send name, expiresInDays, scopes
		if v, ok := got["name"]; !ok || v != "my-token" {
			t.Errorf("request name = %v, want \"my-token\"", v)
		}
		if _, ok := got["tokenName"]; ok {
			t.Error("request should not contain \"tokenName\" field for 2.8.x")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"uuid":"tok-uuid","name":"my-token","token":"jwt-value","expireAt":"2027-01-01T00:00:00Z","scopes":["*"]}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	token, err := client.CreateApiToken(context.Background(), &ApiToken{
		Name:          "my-token",
		ExpiresInDays: 7,
		Scopes:        []string{"*"},
	})
	if err != nil {
		t.Fatalf("CreateApiToken() error = %v", err)
	}

	if token.UUID != "tok-uuid" {
		t.Errorf("UUID = %q, want \"tok-uuid\"", token.UUID)
	}
	if token.Name != "my-token" {
		t.Errorf("Name = %q, want \"my-token\"", token.Name)
	}
	if token.Token != "jwt-value" {
		t.Errorf("Token = %q, want \"jwt-value\"", token.Token)
	}
	if token.ExpireAt != "2027-01-01T00:00:00Z" {
		t.Errorf("ExpireAt = %q, want \"2027-01-01T00:00:00Z\"", token.ExpireAt)
	}
}
