package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPErrorBodiesAreOmittedWithoutCustomHeaders(t *testing.T) {
	t.Parallel()

	t.Run("login password reflection", func(t *testing.T) {
		const password = "reflected-login-password"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"password":"`+password+`"}`)
		}))
		t.Cleanup(server.Close)

		client, err := NewClient(ClientConfig{
			Endpoint: server.URL,
			Username: "admin",
			Password: password,
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.GetSystemHealth(context.Background())
		assertStatusOnlyError(t, err, http.StatusForbidden, password)
		if !strings.Contains(err.Error(), "login failed") {
			t.Errorf("error = %v, want login context", err)
		}
	})

	t.Run("authorization reflection", func(t *testing.T) {
		const token = "reflected-api-token"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, r.Header.Get("Authorization"))
		}))
		t.Cleanup(server.Close)

		client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: token})
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.GetSystemHealth(context.Background())
		assertStatusOnlyError(t, err, http.StatusBadGateway, token, "Bearer "+token)
	})

	t.Run("request body reflection", func(t *testing.T) {
		const payloadSecret = "reflected-request-password"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Errorf("ReadAll(request body): %v", readErr)
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write(body)
		}))
		t.Cleanup(server.Close)

		client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token"})
		if err != nil {
			t.Fatal(err)
		}

		err = client.doRequest(
			context.Background(),
			http.MethodPost,
			"/api/users",
			map[string]string{"password": payloadSecret},
			nil,
		)
		assertStatusOnlyError(t, err, http.StatusUnprocessableEntity, payloadSecret)
	})

	t.Run("transformed response", func(t *testing.T) {
		const token = "transformed-api-token"
		encoded := base64.StdEncoding.EncodeToString([]byte(token))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, encoded)
		}))
		t.Cleanup(server.Close)

		client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: token})
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.GetSystemHealth(context.Background())
		assertStatusOnlyError(t, err, http.StatusInternalServerError, token, encoded)
	})

	t.Run("body is not read", func(t *testing.T) {
		readErr := errors.New("body read disclosed a secret")
		client, err := NewClient(ClientConfig{
			Endpoint: "http://example.test",
			APIToken: "static-token",
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
		assertStatusOnlyError(t, err, http.StatusBadGateway, readErr.Error())
		if errors.Is(err, readErr) {
			t.Fatal("HTTP error retained the response-body read error")
		}
	})
}

func assertStatusOnlyError(t *testing.T, err error, status int, forbidden ...string) {
	t.Helper()

	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %v, want *HTTPStatusError", err)
	}
	if statusErr.StatusCode != status {
		t.Errorf("status = %d, want %d", statusErr.StatusCode, status)
	}
	if statusErr.Body != "" {
		t.Errorf("HTTPStatusError.Body = %q, want empty", statusErr.Body)
	}
	for _, value := range forbidden {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Errorf("error disclosed %q: %v", value, err)
		}
	}
}
