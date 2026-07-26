package provider

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeResponseBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("just below limit", func(t *testing.T) {
		resp := responseWithBody(http.StatusOK, io.LimitReader(zeroReader{}, maxResponseBodyBytes-1))
		if err := (&Client{}).decodeResponse(resp, nil); err != nil {
			t.Fatalf("decodeResponse() error = %v", err)
		}
	})

	t.Run("just above limit", func(t *testing.T) {
		const secret = "oversized-response-secret"
		body := io.MultiReader(
			io.LimitReader(zeroReader{}, maxResponseBodyBytes+1),
			strings.NewReader(secret),
		)
		err := (&Client{}).decodeResponse(responseWithBody(http.StatusOK, body), nil)
		assertOversizedResponseError(t, err, secret)
	})

	t.Run("error body is not read", func(t *testing.T) {
		body := &trackingReadCloser{
			Reader: io.LimitReader(zeroReader{}, maxResponseBodyBytes+1),
		}
		err := (&Client{}).decodeResponse(&http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       body,
		}, nil)
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadGateway {
			t.Fatalf("decodeResponse() error = %v, want status-only 502 error", err)
		}
		if body.read {
			t.Fatal("error response body was read")
		}
	})
}

func TestDecodeResponseRejectsEveryNon2xxWithoutReadingBody(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusSwitchingProtocols,
		199,
		http.StatusMultipleChoices,
		http.StatusNotModified,
		http.StatusBadRequest,
		http.StatusInternalServerError,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := &trackingReadCloser{Reader: strings.NewReader("reflected-response-secret")}
			err := (&Client{}).decodeResponse(&http.Response{
				StatusCode: status,
				Body:       body,
			}, nil)

			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != status {
				t.Fatalf("decodeResponse() error = %v, want status-only %d error", err, status)
			}
			if body.read {
				t.Fatal("non-2xx response body was read")
			}
			if strings.Contains(err.Error(), "reflected-response-secret") {
				t.Fatalf("non-2xx error disclosed response body: %v", err)
			}
		})
	}
}

func TestRedirectWithoutLocationIsAnHTTPError(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "reflected-redirect-secret")
			}))
			t.Cleanup(server.Close)

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetSystemHealth(context.Background())
			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != status {
				t.Fatalf("GetSystemHealth() error = %v, want status-only %d error", err, status)
			}
			if strings.Contains(err.Error(), "reflected-redirect-secret") {
				t.Fatalf("redirect error disclosed response body: %v", err)
			}
		})
	}
}

func TestRequestWithNilOutputRejectsNon2xx(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{
		Endpoint: "http://example.test",
		APIToken: "static-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotModified,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("reflected-action-secret")),
			Request:    req,
		}, nil
	})

	err = client.doRequest(context.Background(), http.MethodDelete, "/api/nodes/node-id", nil, nil)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotModified {
		t.Fatalf("doRequest() error = %v, want status-only 304 error", err)
	}
	if strings.Contains(err.Error(), "reflected-action-secret") {
		t.Fatalf("action error disclosed response body: %v", err)
	}
}

func TestChunkedResponseBodyLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}
		_, _ = w.Write(make([]byte, 1))
		flusher.Flush()
		_, _ = io.CopyN(w, zeroReader{}, maxResponseBodyBytes)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemHealth(context.Background())
	assertOversizedResponseError(t, err)
}

func TestCompressedResponseBodyLimitAppliesAfterDecompression(t *testing.T) {
	t.Parallel()

	const secret = "compressed-oversized-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = io.CopyN(gz, zeroReader{}, maxResponseBodyBytes+1)
		_, _ = io.WriteString(gz, secret)
		_ = gz.Close()
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "static-token"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemHealth(context.Background())
	assertOversizedResponseError(t, err, secret)
}

func TestInfiniteResponseReaderStopsAtLimit(t *testing.T) {
	t.Parallel()

	reader := &countingZeroReadCloser{}
	err := (&Client{}).decodeResponse(&http.Response{
		StatusCode: http.StatusOK,
		Body:       reader,
	}, nil)
	assertOversizedResponseError(t, err)
	if reader.bytesRead != maxResponseBodyBytes+1 {
		t.Fatalf("bytes read = %d, want %d", reader.bytesRead, maxResponseBodyBytes+1)
	}
}

func responseWithBody(status int, body io.Reader) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(body),
	}
}

func assertOversizedResponseError(t *testing.T, err error, forbidden ...string) {
	t.Helper()

	if !errors.Is(err, errResponseBodyTooLarge) {
		t.Fatalf("error = %v, want %v", err, errResponseBodyTooLarge)
	}
	for _, value := range forbidden {
		if strings.Contains(err.Error(), value) {
			t.Errorf("error disclosed %q: %v", value, err)
		}
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type trackingReadCloser struct {
	io.Reader
	read bool
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	r.read = true
	return r.Reader.Read(p)
}

func (*trackingReadCloser) Close() error { return nil }

type countingZeroReadCloser struct {
	bytesRead int64
}

func (r *countingZeroReadCloser) Read(p []byte) (int, error) {
	clear(p)
	r.bytesRead += int64(len(p))
	return len(p), nil
}

func (*countingZeroReadCloser) Close() error { return nil }
