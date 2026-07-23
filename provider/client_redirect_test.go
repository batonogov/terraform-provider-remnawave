package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRedirectPolicyWithoutCustomHeaders(t *testing.T) {
	t.Parallel()

	t.Run("rejects origin changes", func(t *testing.T) {
		tests := []struct {
			name     string
			endpoint string
			location string
		}{
			{name: "hostname", endpoint: "http://example.test", location: "http://other.test/target"},
			{name: "effective port", endpoint: "http://example.test:8080", location: "http://example.test:8081/target"},
			{name: "scheme downgrade", endpoint: "https://example.test", location: "http://example.test/target"},
			{name: "subdomain", endpoint: "https://example.test", location: "https://sub.example.test/target"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				const token = "redirect-secret-token"
				client, err := NewClient(ClientConfig{Endpoint: tt.endpoint, APIToken: token})
				if err != nil {
					t.Fatal(err)
				}

				var roundTrips atomic.Int32
				client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
					roundTrips.Add(1)
					return redirectResponse(req, tt.location), nil
				})

				_, err = client.GetSystemHealth(context.Background())
				if !errors.Is(err, errCrossOriginRedirect) {
					t.Fatalf("GetSystemHealth() error = %v, want cross-origin redirect error", err)
				}
				if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), tt.location) {
					t.Fatalf("redirect error disclosed request data: %v", err)
				}
				if roundTrips.Load() != 1 {
					t.Errorf("round trips = %d, want 1", roundTrips.Load())
				}
			})
		}
	})

	t.Run("rejects login body replay", func(t *testing.T) {
		const password = "redirect-login-password"
		client, err := NewClient(ClientConfig{
			Endpoint: "https://example.test",
			Username: "admin",
			Password: password,
		})
		if err != nil {
			t.Fatal(err)
		}

		var roundTrips atomic.Int32
		client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			roundTrips.Add(1)
			response := redirectResponse(req, "https://other.test/login")
			response.StatusCode = http.StatusPermanentRedirect
			return response, nil
		})

		err = client.authenticate(context.Background())
		if !errors.Is(err, errCrossOriginRedirect) {
			t.Fatalf("authenticate() error = %v, want cross-origin redirect error", err)
		}
		if strings.Contains(err.Error(), password) {
			t.Fatalf("redirect error disclosed login password: %v", err)
		}
		if roundTrips.Load() != 1 {
			t.Errorf("round trips = %d, want 1", roundTrips.Load())
		}
	})

	t.Run("rejects mutating request body replay", func(t *testing.T) {
		const payloadSecret = "redirect-payload-secret"
		client, err := NewClient(ClientConfig{
			Endpoint: "https://example.test",
			APIToken: "redirect-api-token",
		})
		if err != nil {
			t.Fatal(err)
		}

		var roundTrips atomic.Int32
		client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			roundTrips.Add(1)
			response := redirectResponse(req, "https://other.test/users")
			response.StatusCode = http.StatusTemporaryRedirect
			return response, nil
		})

		err = client.doRequest(
			context.Background(),
			http.MethodPost,
			"/api/users",
			map[string]string{"password": payloadSecret},
			nil,
		)
		if !errors.Is(err, errCrossOriginRedirect) {
			t.Fatalf("doRequest() error = %v, want cross-origin redirect error", err)
		}
		if strings.Contains(err.Error(), payloadSecret) {
			t.Fatalf("redirect error disclosed request payload: %v", err)
		}
		if roundTrips.Load() != 1 {
			t.Errorf("round trips = %d, want 1", roundTrips.Load())
		}
	})

	t.Run("accepts normalized default port", func(t *testing.T) {
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
					Endpoint: tt.endpoint,
					APIToken: "same-origin-token",
				})
				if err != nil {
					t.Fatal(err)
				}

				var roundTrips atomic.Int32
				client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if roundTrips.Add(1) == 1 {
						return redirectResponse(req, tt.location), nil
					}
					if got := req.Header.Get("Authorization"); got != "Bearer same-origin-token" {
						t.Errorf("redirected Authorization = %q", got)
					}
					return jsonResponse(req, http.StatusOK, `{"response":{"status":"ok"}}`), nil
				})

				if _, err := client.GetSystemHealth(context.Background()); err != nil {
					t.Fatalf("GetSystemHealth() error = %v", err)
				}
				if roundTrips.Load() != 2 {
					t.Errorf("round trips = %d, want 2", roundTrips.Load())
				}
			})
		}
	})
}
