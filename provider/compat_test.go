package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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
		{"v3.0.1", "3.0"},
		{"1", ""},
		{"", ""},
		{"garbage", ""},
		{"v2.8.0", "2.8"},
		{"abc.def", ""},
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
	if v3, err := client.isVersionAtLeast3_0(context.Background()); err != nil || v3 {
		t.Errorf("isVersionAtLeast3_0() = %v, %v, want false, nil", v3, err)
	}
}

func TestVersionDetection3_0(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"version":"3.0.0"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	v3, err := client.isVersionAtLeast3_0(context.Background())
	if err != nil || !v3 {
		t.Errorf("isVersionAtLeast3_0() = %v, %v, want true, nil", v3, err)
	}
	v31, err := client.isVersionAtLeast3_1(context.Background())
	if err != nil || v31 {
		t.Errorf("isVersionAtLeast3_1() = %v, %v, want false, nil", v31, err)
	}
}

func TestVersionDetection3_1(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"version":"3.1.0"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]func(context.Context) (bool, error){
		"3.0": client.isVersionAtLeast3_0,
		"3.1": client.isVersionAtLeast3_1,
	} {
		got, err := check(context.Background())
		if err != nil || !got {
			t.Errorf("isVersionAtLeast%s() = %v, %v, want true, nil", name, got, err)
		}
	}
}

func TestVersionDetection3_2_2(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		version string
		want    bool
	}{
		{version: "3.2.1", want: false},
		{version: "3.2.2", want: true},
		{version: "3.3.0", want: true},
	} {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"response":{"version":"`+tt.version+`"}}`)
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.isVersionAtLeast3_2_2(context.Background())
			if err != nil || got != tt.want {
				t.Errorf("isVersionAtLeast3_2_2() = %v, %v, want %v, nil", got, err, tt.want)
			}
		})
	}
}

func TestVersionDetection3_2_3(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		version string
		want    bool
		wantErr bool
	}{
		{version: "3.2.2", want: false},
		{version: "3.2.2+build.9", want: false},
		{version: "3.2.3", want: true},
		{version: "v3.2.3+build.7", want: true},
		{version: "3.3.0", want: true},
		{version: "3.2.3-rc.1", wantErr: true},
		{version: "3.2.3+", wantErr: true},
		{version: "3.2.3+build..7", wantErr: true},
	} {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"response":{"version":"`+tt.version+`"}}`)
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.isVersionAtLeast3_2_3(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("isVersionAtLeast3_2_3() = %v, nil, want error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("isVersionAtLeast3_2_3() = %v, %v, want %v, nil", got, err, tt.want)
			}
		})
	}
}

func TestVersionDetection3_3_1(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		version string
		want    bool
		wantErr bool
	}{
		{version: "3.3.0", want: false},
		{version: "3.3.0+build.4", want: false},
		{version: "3.3.1", want: true},
		{version: "v3.3.2", want: true},
		{version: "3.4.0", want: true},
		{version: "3.3.1-rc.1", wantErr: true},
	} {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"response":{"version":"`+tt.version+`"}}`)
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.isVersionAtLeast3_3_1(t.Context())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("isVersionAtLeast3_3_1() = %v, nil, want error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("isVersionAtLeast3_3_1() = %v, %v, want %v, nil", got, err, tt.want)
			}
		})
	}
}

func TestVersionDetection3_3(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		version string
		want    bool
	}{
		{version: "3.2.3", want: false},
		{version: "3.3.0", want: true},
		{version: "3.4.0", want: true},
	} {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"response":{"version":"`+tt.version+`"}}`)
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.isVersionAtLeast3_3(t.Context())
			if err != nil || got != tt.want {
				t.Errorf("isVersionAtLeast3_3() = %v, %v, want %v, nil", got, err, tt.want)
			}
		})
	}
}

func TestVersionDetectionRejectsOversizedMetadata(t *testing.T) {
	t.Parallel()

	version := "3.2." + strings.Repeat("9", 1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"response": map[string]string{"version": version}})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.isVersionAtLeast3_2_2(t.Context())
	if err == nil {
		t.Fatal("isVersionAtLeast3_2_2() error = nil")
	}
	if len(err.Error()) > 200 {
		t.Fatalf("version error length = %d, want <= 200", len(err.Error()))
	}
	client.versionMu.Lock()
	cachedLength := len(client.serverFullVersion)
	client.versionMu.Unlock()
	if cachedLength > 64 {
		t.Fatalf("cached version length = %d, want <= 64", cachedLength)
	}
}

func TestHostRequestV27UsesSingularTag(t *testing.T) {
	t.Parallel()

	var hostRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/system/metadata" {
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		hostRequests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode host request: %v", err)
		}
		if got := body["tag"]; got != "LEGACY" {
			t.Errorf("request tag = %v, want LEGACY", got)
		}
		if _, ok := body["tags"]; ok {
			t.Error("2.7 request must not contain plural tags")
		}
		_, _ = io.WriteString(w, `{"response":{"uuid":"host-id","remark":"host","address":"host.example.com","port":443,"tag":"LEGACY"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := client.CreateHost(context.Background(), &Host{
		Remark: "host", Address: "host.example.com", Port: 443, Tags: new([]string{"LEGACY"}),
	})
	if err != nil {
		t.Fatalf("CreateHost() error = %v", err)
	}
	if host.Tag == nil || *host.Tag != "LEGACY" {
		t.Fatalf("response legacy tag = %#v", host.Tag)
	}

	_, err = client.UpdateHost(context.Background(), &Host{Tags: new([]string{"ONE", "TWO"})})
	if err == nil || !strings.Contains(err.Error(), "at most one host tag") {
		t.Fatalf("UpdateHost() multi-tag error = %v", err)
	}
	if hostRequests != 1 {
		t.Fatalf("host requests = %d, want 1", hostRequests)
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
	if _, err := client.isVersionAtLeast3_0(context.Background()); err == nil {
		t.Error("isVersionAtLeast3_0() error = nil after detection failure")
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

// TestParseMajorMinor_PreReleaseAndBuild extends semver-extraction coverage to
// pre-release/build metadata, optional v prefixes, and malformed versions.
func TestParseMajorMinor_PreReleaseAndBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{in: "2.8.1", want: "2.8"},
		{in: "2.7.4", want: "2.7"},
		{in: "2.8.1-rc.1+build", want: "2.8"},
		{in: "2.10.0", want: "2.10"},
		{in: "2.8", want: "2.8"},
		{in: "2", want: ""},
		{in: "", want: ""},
		{in: "v2.8.0", want: "2.8"},
		{in: "abc.def", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseMajorMinor(tt.in); got != tt.want {
				t.Errorf("parseMajorMinor(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestConcurrentVersionDetectionIsRaceFree verifies that a burst of requests
// on a fresh client triggers exactly one /api/system/metadata call and that
// detection is correctly serialized under the version mutex.
func TestConcurrentVersionDetectionIsRaceFree(t *testing.T) {
	t.Parallel()

	var metadataCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			metadataCalls.Add(1)
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

	const goroutines = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			_ = client.isVersion2_7(context.Background())
		}()
	}
	close(start)
	wg.Wait()

	if got := client.serverVersion; got != "2.7" {
		t.Errorf("serverVersion = %q, want %q", got, "2.7")
	}
	if n := metadataCalls.Load(); n != 1 {
		t.Errorf("/api/system/metadata called %d times, want 1 (must be detected once)", n)
	}
}

// TestVersionDetectionMalformedVersionField verifies that a 200 response with a
// non-semver "version" field is rejected: serverMinorVersion falls back to "",
// isVersion2_7 returns false, and detectVersion surfaces an explicit error.
func TestVersionDetectionMalformedVersionField(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"garbage"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	t.Run("serverMinorVersion and isVersion2_7 fall back", func(t *testing.T) {
		client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
		if err != nil {
			t.Fatal(err)
		}
		if got := client.serverMinorVersion(context.Background()); got != "" {
			t.Errorf("serverMinorVersion() = %q, want empty", got)
		}
		if client.isVersion2_7(context.Background()) {
			t.Error("isVersion2_7() = true, want false for malformed version")
		}
	})

	t.Run("detectVersion surfaces unexpected-format error", func(t *testing.T) {
		client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
		if err != nil {
			t.Fatal(err)
		}
		err = client.detectVersion(context.Background())
		if err == nil {
			t.Fatal("detectVersion() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "unexpected version format") {
			t.Errorf("detectVersion() error = %q, want it to contain unexpected version format", err)
		}
	})
}

// TestVersionDetectionNetworkFailureDefaultsTo28 verifies that a transport
// failure on /api/system/metadata does not crash the client: the error is
// swallowed by serverMinorVersion, version stays empty and isVersion2_7 is
// false (2.8 behaviour), with no leaked error.
func TestVersionDetectionNetworkFailureDefaultsTo28(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{Endpoint: "https://example.test", APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	// Transport returns a network failure for every request (metadata is the
	// only endpoint exercised here). It must not propagate out of
	// serverMinorVersion/isVersion2_7.
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errHTTPRequestFailed
	})

	if got := client.serverMinorVersion(context.Background()); got != "" {
		t.Errorf("serverMinorVersion() = %q, want empty on transport failure", got)
	}
	if client.isVersion2_7(context.Background()) {
		t.Error("isVersion2_7() = true after transport failure, want false")
	}
	if _, err := client.isVersionAtLeast3_0(context.Background()); err == nil {
		t.Error("isVersionAtLeast3_0() error = nil after transport failure")
	}
}

// TestClient_HostRequestV27_ZeroTagsOmitsTagFieldOnUpdate complements the existing
// 1-tag and >1-tag cases by verifying that a host with ZERO tags omits the
// singular "tag" field entirely on the 2.7 PATCH path.
func TestClient_HostRequestV27_ZeroTagsOmitsTagFieldOnUpdate(t *testing.T) {
	t.Parallel()

	var updateBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/system/metadata" {
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		if r.Method == http.MethodPatch && r.URL.Path == "/api/hosts" {
			updateBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, `{"response":{"uuid":"host-id","remark":"host","address":"host.example.com","port":443}}`)
			return
		}
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.UpdateHost(context.Background(), &Host{
		Remark:  "host",
		Address: "host.example.com",
		Port:    443,
		Tags:    nil, // zero tags
	})
	if err != nil {
		t.Fatalf("UpdateHost() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(updateBody, &got); err != nil {
		t.Fatalf("decode PATCH body: %v", err)
	}
	if v, ok := got["tag"]; ok {
		t.Errorf("2.7 PATCH body contained singular tag field with zero tags: %v", v)
	}
	if v, ok := got["tags"]; ok {
		t.Errorf("2.7 PATCH body contained plural tags field: %v", v)
	}
}

// TestClient_CreateApiTokenV27_EmptyScopesDefault verifies the 2.7.x token
// creation path when Scopes is empty: the request uses only tokenName (no
// scopes field is sent) and the returned ApiToken defaults Scopes to ["*"].
func TestClient_CreateApiTokenV27_EmptyScopesDefault(t *testing.T) {
	t.Parallel()

	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/system/metadata" {
			_, _ = io.WriteString(w, `{"response":{"version":"2.7.4"}}`)
			return
		}
		if r.URL.Path == "/api/tokens" && r.Method == http.MethodPost {
			requestBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, `{"response":{"uuid":"tok-uuid","token":"jwt-value","tokenName":"scoped"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	token, err := client.CreateApiToken(context.Background(), &ApiToken{
		Name:   "scoped",
		Scopes: nil, // empty scopes
	})
	if err != nil {
		t.Fatalf("CreateApiToken() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(requestBody, &got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	// 2.7 sends only tokenName; scopes must not appear in the request body.
	if v, ok := got["tokenName"]; !ok || v != "scoped" {
		t.Errorf("request tokenName = %v, want scoped", v)
	}
	if v, ok := got["scopes"]; ok {
		t.Errorf("2.7 request must not contain scopes field: %v", v)
	}

	// The client hard-codes Scopes to ["*"] for 2.7.x responses.
	if len(token.Scopes) != 1 || token.Scopes[0] != "*" {
		t.Errorf("token Scopes = %v, want [*]", token.Scopes)
	}
	if token.UUID != "tok-uuid" || token.Token != "jwt-value" {
		t.Errorf("token = %+v, want uuid=tok-uuid token=jwt-value", token)
	}
}
