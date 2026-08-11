package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestWaitForRemnawaveAPIReady(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/status" {
			t.Errorf("path = %q, want /api/auth/status", r.URL.Path)
		}
		if got := r.Header.Get("X-Forwarded-For"); got != "127.0.0.1" {
			t.Errorf("X-Forwarded-For = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "https" {
			t.Errorf("X-Forwarded-Proto = %q", got)
		}
		if got := r.Header.Get("X-Remnawave-Client-Type"); got != "browser" {
			t.Errorf("X-Remnawave-Client-Type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"isRegistrationAllowed":true}}`))
	}))
	defer server.Close()

	output, err := runWaitForRemnawaveAPI(t, t.Context(), server.URL,
		"REMNAWAVE_READY_ATTEMPTS=3",
		"REMNAWAVE_READY_DELAY_SECONDS=0",
		"REMNAWAVE_READY_TIMEOUT_SECONDS=3",
	)
	if err != nil {
		t.Fatalf("wait helper error = %v, output = %s", err, output)
	}
}

func TestWaitForRemnawaveAPITotalDeadline(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	started := time.Now()
	output, err := runWaitForRemnawaveAPI(t, ctx, server.URL,
		"REMNAWAVE_READY_ATTEMPTS=20",
		"REMNAWAVE_READY_DELAY_SECONDS=0",
		"REMNAWAVE_READY_TIMEOUT_SECONDS=1",
	)
	elapsed := time.Since(started)
	if ctx.Err() != nil {
		t.Fatalf("wait helper ignored its one-second deadline: %v (output: %s)", elapsed, output)
	}
	if err == nil {
		t.Fatal("wait helper error = nil, want timeout")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("wait helper elapsed = %v, want <= 3s", elapsed)
	}
	if !strings.Contains(output, "within 1 seconds") {
		t.Fatalf("wait helper output = %q, want bounded-timeout diagnostic", output)
	}
}

func TestWaitForRemnawaveAPIRejectsInvalidBounds(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		env  string
		want string
	}{
		{name: "zero attempts", env: "REMNAWAVE_READY_ATTEMPTS=0", want: "REMNAWAVE_READY_ATTEMPTS"},
		{name: "excessive attempts", env: "REMNAWAVE_READY_ATTEMPTS=301", want: "REMNAWAVE_READY_ATTEMPTS"},
		{name: "non numeric delay", env: "REMNAWAVE_READY_DELAY_SECONDS=forever", want: "REMNAWAVE_READY_DELAY_SECONDS"},
		{name: "excessive timeout", env: "REMNAWAVE_READY_TIMEOUT_SECONDS=301", want: "REMNAWAVE_READY_TIMEOUT_SECONDS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output, err := runWaitForRemnawaveAPI(t, t.Context(), "http://127.0.0.1:1", test.env)
			if err == nil {
				t.Fatal("wait helper error = nil, want validation error")
			}
			if !strings.Contains(output, test.want) {
				t.Fatalf("wait helper output = %q, want %q", output, test.want)
			}
		})
	}
}

func runWaitForRemnawaveAPI(t *testing.T, ctx context.Context, endpoint string, env ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "../scripts/wait-for-remnawave-api.sh", endpoint)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
