package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client manages communication with the Remnawave REST API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiToken   string

	authMu      sync.Mutex
	accessToken string
	tokenExpiry time.Time
	username    string
	password    string
}

// ClientConfig holds the parameters for creating a new Client.
type ClientConfig struct {
	Endpoint           string
	APIToken           string
	Username           string
	Password           string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

type apiResponse struct {
	Response json.RawMessage `json:"response"`
}

// HTTPStatusError carries the HTTP status code and body from a failed request.
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("request failed: status %d", e.StatusCode)
	}
	return fmt.Sprintf("request failed: status %d, body: %s", e.StatusCode, e.Body)
}

// NewClient creates a new Remnawave API client.
// If APIToken is provided, it is used as a Bearer token directly (no login needed).
// Otherwise, username/password login is used to obtain a JWT.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	baseURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	if baseURL.Scheme == "" {
		return nil, errors.New("endpoint must include scheme (http or https)")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// #nosec G402 -- InsecureSkipVerify is intentional: self-hosted panels frequently use self-signed certificates.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		apiToken:   cfg.APIToken,
		username:   cfg.Username,
		password:   cfg.Password,
	}, nil
}

// authenticate obtains a JWT access token via username/password login,
// unless an API token is already configured.
func (c *Client) authenticate(ctx context.Context) error {
	if c.apiToken != "" {
		return nil // static API token — no login needed
	}
	if c.username == "" || c.password == "" {
		return errors.New("either api_token or username+password must be configured")
	}

	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	var resp struct {
		Response struct {
			AccessToken string `json:"accessToken"`
		} `json:"response"`
	}
	if err := c.doRaw(ctx, http.MethodPost, "/api/auth/login", payload, &resp); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if resp.Response.AccessToken == "" {
		return errors.New("login succeeded but no access token returned")
	}

	c.authMu.Lock()
	c.accessToken = resp.Response.AccessToken
	// JWT lifetime is configurable on the panel (default 12h). We use 11h
	// as a conservative default to refresh before expiry.
	c.tokenExpiry = time.Now().Add(11 * time.Hour)
	c.authMu.Unlock()
	return nil
}

// token returns the current bearer token, refreshing if needed.
func (c *Client) token(ctx context.Context) (string, error) {
	if c.apiToken != "" {
		return c.apiToken, nil
	}

	c.authMu.Lock()
	expired := time.Now().After(c.tokenExpiry) || c.accessToken == ""
	token := c.accessToken
	c.authMu.Unlock()

	if expired {
		if err := c.authenticate(ctx); err != nil {
			return "", err
		}
		c.authMu.Lock()
		token = c.accessToken
		c.authMu.Unlock()
	}
	return token, nil
}

// doRaw sends an HTTP request WITHOUT authentication headers.
// Used by authenticate() to avoid infinite recursion.
func (c *Client) doRaw(ctx context.Context, method, path string, body any, out any) error {
	endpoint := c.resolvePath(path)

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.sendRequest(req, out)
}

// doRequest sends an authenticated JSON request and decodes the response.
func (c *Client) doRequest(ctx context.Context, method, path string, body any, out any) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	endpoint := c.resolvePath(path)

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// On 401, try re-authenticating once (unless using static API token).
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized && c.apiToken == "" {
		// #nosec G104 -- discarding body before re-auth; Close error is not actionable
		resp.Body.Close() //nolint:errcheck // best-effort close before re-auth
		c.authMu.Lock()
		c.accessToken = ""
		c.authMu.Unlock()
		token, err = c.token(ctx)
		if err != nil {
			return err
		}
		req2, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
		if err != nil {
			return err
		}
		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("Content-Type", "application/json")
		// Reset body reader for retry
		if body != nil {
			b, _ := json.Marshal(body)
			req2.Body = io.NopCloser(bytes.NewReader(b))
		}
		resp, err = c.httpClient.Do(req2)
		if err != nil {
			return err
		}
	}

	return c.decodeResponse(resp, out)
}

func (c *Client) sendRequest(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	return c.decodeResponse(resp, out)
}

func (c *Client) decodeResponse(resp *http.Response, out any) error {
	// #nosec G104 -- discarding body close error; not actionable after read
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 1024 {
			msg = msg[:1024] + "...(truncated)"
		}
		return &HTTPStatusError{StatusCode: resp.StatusCode, Body: msg}
	}

	if out == nil || len(body) == 0 {
		return nil
	}

	// Remnawave wraps responses in { "response": <data> }
	var envelope apiResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		// Maybe it's a raw value (some endpoints return raw)
		return json.Unmarshal(body, out)
	}
	if len(envelope.Response) > 0 {
		return json.Unmarshal(envelope.Response, out)
	}
	// Fallback: try direct decode
	return json.Unmarshal(body, out)
}

func (c *Client) resolvePath(path string) string {
	base := *c.baseURL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + path
	return base.String()
}

// ─── User API ───

func (c *Client) CreateUser(ctx context.Context, user *User) (*User, error) {
	var out User
	if err := c.doRequest(ctx, http.MethodPost, "/api/users", user, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetUserByUUID(ctx context.Context, uuid string) (*User, error) {
	var out User
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/users/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(ctx context.Context, user *User) (*User, error) {
	var out User
	if err := c.doRequest(ctx, http.MethodPatch, "/api/users", user, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/users/%s", uuid), nil, nil)
}

// ─── Node API ───

func (c *Client) CreateNode(ctx context.Context, node *Node) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodPost, "/api/nodes", node, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAllNodes(ctx context.Context) ([]Node, error) {
	var out []Node
	if err := c.doRequest(ctx, http.MethodGet, "/api/nodes", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetNodeByUUID(ctx context.Context, uuid string) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/nodes/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateNode(ctx context.Context, node *Node) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodPatch, "/api/nodes", node, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteNode(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/nodes/%s", uuid), nil, nil)
}

// ─── Host API ───

func (c *Client) CreateHost(ctx context.Context, host *Host) (*Host, error) {
	var out Host
	if err := c.doRequest(ctx, http.MethodPost, "/api/hosts", host, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAllHosts(ctx context.Context) ([]Host, error) {
	var out []Host
	if err := c.doRequest(ctx, http.MethodGet, "/api/hosts", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetHostByUUID(ctx context.Context, uuid string) (*Host, error) {
	var out Host
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/hosts/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateHost(ctx context.Context, host *Host) (*Host, error) {
	var out Host
	if err := c.doRequest(ctx, http.MethodPatch, "/api/hosts", host, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteHost(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/hosts/%s", uuid), nil, nil)
}

// ─── System API ───

func (c *Client) GetSystemHealth(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/health", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
