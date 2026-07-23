package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestUnauthorizedReplayPolicyByMethod(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			const (
				password     = "not-in-diagnostic-password"
				headerSecret = "not-in-diagnostic-header"
				bodySecret   = "not-in-diagnostic-body"
				jwtSecret    = "not-in-diagnostic-jwt"
			)
			var mu sync.Mutex
			loginCalls := 0
			apiCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				switch r.URL.Path {
				case "/api/auth/login":
					loginCalls++
					_, _ = fmt.Fprintf(w, `{"response":{"accessToken":"%s"}}`, jwtSecret)
				case "/api/test":
					apiCalls++
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = fmt.Fprintf(
						w,
						`{"reflected":["%s","%s","%s","%s"]}`,
						password,
						headerSecret,
						bodySecret,
						jwtSecret,
					)
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			client, err := NewClient(ClientConfig{
				Endpoint:      server.URL,
				Username:      "admin",
				Password:      password,
				CustomHeaders: map[string]string{"X-Gateway-Token": headerSecret},
			})
			if err != nil {
				t.Fatal(err)
			}
			err = client.doRequest(
				context.Background(),
				method,
				"/api/test",
				map[string]string{"secret": bodySecret},
				nil,
			)
			if !errors.Is(err, errMutatingRequestUnauthorized) {
				t.Fatalf("doRequest() error = %v, want %v", err, errMutatingRequestUnauthorized)
			}
			for _, secret := range []string{password, headerSecret, bodySecret, jwtSecret} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("error disclosed %q: %v", secret, err)
				}
			}
			if loginCalls != 1 || apiCalls != 1 {
				t.Errorf("calls: login=%d api=%d, want login=1 api=1", loginCalls, apiCalls)
			}
		})
	}
}

func TestUnauthorizedActionPayloadIsNotReplayed(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			_, _ = io.WriteString(w, `{"response":{"accessToken":"login-token"}}`)
		case "/api/nodes/node-id/actions/restart":
			calls++
			w.WriteHeader(http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		Endpoint: server.URL,
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.RestartNode(context.Background(), "node-id", true)
	if !errors.Is(err, errMutatingRequestUnauthorized) {
		t.Fatalf("RestartNode() error = %v, want %v", err, errMutatingRequestUnauthorized)
	}
	if calls != 1 {
		t.Fatalf("restart calls = %d, want 1", calls)
	}
}

func TestHeadUnauthorizedIsRetried(t *testing.T) {
	t.Parallel()

	var loginCalls, headCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			loginCalls++
			_, _ = fmt.Fprintf(w, `{"response":{"accessToken":"token-%d"}}`, loginCalls)
		case "/api/test":
			headCalls++
			if r.Method != http.MethodHead {
				t.Errorf("method = %s, want HEAD", r.Method)
			}
			if headCalls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		Endpoint: server.URL,
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.doRequest(context.Background(), http.MethodHead, "/api/test", nil, nil); err != nil {
		t.Fatalf("HEAD error = %v", err)
	}
	if loginCalls != 2 || headCalls != 2 {
		t.Fatalf("calls: login=%d head=%d, want 2 each", loginCalls, headCalls)
	}
}
