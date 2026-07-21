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
	"strconv"
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

	proxyHeaders bool

	// serverVersion is the major.minor of the Remnawave backend (e.g. "2.7",
	// "2.8"), detected lazily on the first request via /api/system/metadata.
	// A value of "" means detection has not yet been attempted.
	versionMu     sync.Mutex
	serverVersion string
}

// ClientConfig holds the parameters for creating a new Client.
type ClientConfig struct {
	Endpoint           string
	APIToken           string
	Username           string
	Password           string
	InsecureSkipVerify bool
	Timeout            time.Duration
	// ProxyHeaders adds X-Forwarded-For and X-Forwarded-Proto headers
	// to every request. Needed when connecting directly to the panel
	// without a reverse proxy (e.g. acceptance tests).
	ProxyHeaders bool
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
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, errors.New("endpoint scheme must be http or https")
	}
	if baseURL.Host == "" {
		return nil, errors.New("endpoint must include a host")
	}
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, errors.New("endpoint must not include a query string or fragment")
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
		baseURL:      baseURL,
		httpClient:   httpClient,
		apiToken:     cfg.APIToken,
		username:     cfg.Username,
		password:     cfg.Password,
		proxyHeaders: cfg.ProxyHeaders,
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

	// Serialize authentication and check the token again after acquiring the
	// lock. Without the second check, a burst of initial requests can trigger
	// one login per request.
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	// decodeResponse unwraps the { "response": <data> } envelope,
	// so out receives <data> directly (i.e. { "accessToken": "..." }).
	var resp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := c.doRaw(ctx, http.MethodPost, "/api/auth/login", payload, &resp); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if resp.AccessToken == "" {
		return errors.New("login succeeded but no access token returned")
	}

	c.accessToken = resp.AccessToken
	// JWT lifetime is configurable on the panel (default 12h). We use 11h
	// as a conservative default to refresh before expiry.
	c.tokenExpiry = time.Now().Add(11 * time.Hour)
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
	c.setProxyHeaders(req)

	return c.sendRequest(req, out)
}

// doRequest sends an authenticated JSON request and decodes the response.
func (c *Client) doRequest(ctx context.Context, method, path string, body any, out any) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	endpoint := c.resolvePath(path)

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	newRequest := func(token string) (*http.Request, error) {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		if c.apiToken == "" {
			// The panel distinguishes browser sessions from API-token requests.
			// Login-issued admin JWTs are rejected by ProxyCheckMiddleware unless
			// this header is present.
			req.Header.Set("X-Remnawave-Client-Type", "browser")
		}
		c.setProxyHeaders(req)
		return req, nil
	}

	req, err := newRequest(token)
	if err != nil {
		return err
	}

	// On 401, try re-authenticating once (unless using static API token).
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized && c.apiToken == "" {
		// #nosec G104 -- discarding body before re-auth; not actionable
		_ = resp.Body.Close()
		c.authMu.Lock()
		// Do not discard a token that another request has already refreshed.
		if c.accessToken == token {
			c.accessToken = ""
			c.tokenExpiry = time.Time{}
		}
		c.authMu.Unlock()
		token, err = c.token(ctx)
		if err != nil {
			return err
		}
		req2, err := newRequest(token)
		if err != nil {
			return err
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
	defer func() {
		// #nosec G104 -- body close error after read is not actionable
		_ = resp.Body.Close()
	}()

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

	// Handle query string if present in path
	rawQuery := ""
	if idx := strings.Index(path, "?"); idx >= 0 {
		rawQuery = path[idx+1:]
		path = path[:idx]
	}

	base.Path = strings.TrimSuffix(base.Path, "/") + path
	if rawQuery != "" {
		base.RawQuery = rawQuery
	}
	return base.String()
}

// setProxyHeaders adds reverse-proxy headers required by the panel's
// ProxyCheckMiddleware when connecting without a real reverse proxy.
func (c *Client) setProxyHeaders(req *http.Request) {
	if c.proxyHeaders {
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.Header.Set("X-Forwarded-Proto", "https")
	}
}

// ─── Version detection ───

// detectVersion queries /api/system/metadata and caches the server's
// major.minor version (e.g. "2.7", "2.8"). It is called lazily on the
// first API-token operation and is safe to call concurrently.
func (c *Client) detectVersion(ctx context.Context) error {
	c.versionMu.Lock()
	defer c.versionMu.Unlock()
	if c.serverVersion != "" {
		return nil
	}

	var resp struct {
		Version string `json:"version"`
	}
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/metadata", nil, &resp); err != nil {
		return fmt.Errorf("failed to detect server version: %w", err)
	}

	minor := parseMajorMinor(resp.Version)
	if minor == "" {
		return fmt.Errorf("unexpected version format from server: %q", resp.Version)
	}
	c.serverVersion = minor
	return nil
}

// serverMinorVersion returns the cached major.minor, detecting if needed.
func (c *Client) serverMinorVersion(ctx context.Context) string {
	c.versionMu.Lock()
	v := c.serverVersion
	c.versionMu.Unlock()
	if v != "" {
		return v
	}
	_ = c.detectVersion(ctx)
	c.versionMu.Lock()
	defer c.versionMu.Unlock()
	return c.serverVersion
}

// isVersion2_7 returns true if the connected backend reports version 2.7.x.
func (c *Client) isVersion2_7(ctx context.Context) bool {
	return c.serverMinorVersion(ctx) == "2.7"
}

// parseMajorMinor extracts "major.minor" from a semver-like string.
// e.g. "2.7.4" → "2.7", "2.8.0" → "2.8", "garbage" → "".
func parseMajorMinor(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "." + parts[1]
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

func (c *Client) UpdateUser(ctx context.Context, user any) (*User, error) {
	var out User
	if err := c.doRequest(ctx, http.MethodPatch, "/api/users", user, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/users/%s", uuid), nil, nil)
}

type usersListResponse struct {
	Total int    `json:"total"`
	Users []User `json:"users"`
}

func (c *Client) GetAllUsers(ctx context.Context) ([]User, error) {
	var out usersListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/users?size=1000", nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

// ─── User Actions API ───

// userActionEndpoint maps a user action string to its REST endpoint suffix.
// "reset-traffic" is accepted as a backward-compatible alias for the canonical
// "reset_traffic" form (both resolve to the same backend endpoint).
var userActionEndpoint = map[string]string{
	"enable":              "enable",
	"disable":             "disable",
	"reset_traffic":       "reset-traffic",
	"reset-traffic":       "reset-traffic", // backward-compatible alias
	"revoke_subscription": "revoke",
}

// UserAction performs an imperative action (enable, disable, reset_traffic,
// revoke_subscription) on a user via POST /api/users/:uuid/actions/:action.
// The action "reset-traffic" is accepted as an alias for "reset_traffic".
func (c *Client) UserAction(ctx context.Context, userUUID, action string) error {
	suffix, ok := userActionEndpoint[action]
	if !ok {
		return fmt.Errorf("unknown user action %q: must be one of enable, disable, reset_traffic, revoke_subscription", action)
	}
	path := fmt.Sprintf("/api/users/%s/actions/%s", userUUID, suffix)
	return c.doRequest(ctx, http.MethodPost, path, nil, nil)
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

// ─── Node Actions API ───

// EnableNode enables (un-disables) a node.
// POST /api/nodes/:uuid/actions/enable
func (c *Client) EnableNode(ctx context.Context, uuid string) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/nodes/%s/actions/enable", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DisableNode administratively disables a node.
// POST /api/nodes/:uuid/actions/disable
func (c *Client) DisableNode(ctx context.Context, uuid string) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/nodes/%s/actions/disable", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RestartNode restarts the Xray backend on a node.
// POST /api/nodes/:uuid/actions/restart  (body: {"forceRestart": bool})
func (c *Client) RestartNode(ctx context.Context, uuid string, forceRestart bool) (*Node, error) {
	var out Node
	body := map[string]bool{"forceRestart": forceRestart}
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/nodes/%s/actions/restart", uuid), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetNodeTraffic resets the traffic counter for a node.
// POST /api/nodes/:uuid/actions/reset-traffic
func (c *Client) ResetNodeTraffic(ctx context.Context, uuid string) (*Node, error) {
	var out Node
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/nodes/%s/actions/reset-traffic", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── Host API ───

func (c *Client) hostRequest(ctx context.Context, host *Host) (*Host, error) {
	if !c.isVersion2_7(ctx) {
		return host, nil
	}
	if len(host.Tags) > 1 {
		return nil, fmt.Errorf("remnawave 2.7 supports at most one host tag, got %d", len(host.Tags))
	}

	legacy := *host
	legacy.Tags = nil
	legacy.Tag = nil
	if len(host.Tags) == 1 {
		tag := host.Tags[0]
		legacy.Tag = &tag
	}
	return &legacy, nil
}

func (c *Client) CreateHost(ctx context.Context, host *Host) (*Host, error) {
	payload, err := c.hostRequest(ctx, host)
	if err != nil {
		return nil, err
	}
	var out Host
	if err := c.doRequest(ctx, http.MethodPost, "/api/hosts", payload, &out); err != nil {
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
	payload, err := c.hostRequest(ctx, host)
	if err != nil {
		return nil, err
	}
	var out Host
	if err := c.doRequest(ctx, http.MethodPatch, "/api/hosts", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteHost(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/hosts/%s", uuid), nil, nil)
}

// GetHostTags returns all unique host tags.

// hostTagsResponse is the unwrapped {tags: [...]} payload after decodeResponse
// strips the outer {response: ...} envelope.
type hostTagsResponse struct {
	Tags []string `json:"tags"`
}

func (c *Client) GetHostTags(ctx context.Context) ([]string, error) {
	var out hostTagsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/hosts/tags", nil, &out); err != nil {
		return nil, err
	}
	return out.Tags, nil
}

// BulkEnableHosts enables the hosts identified by the given UUIDs.
func (c *Client) BulkEnableHosts(ctx context.Context, uuids []string) error {
	return c.doRequest(ctx, http.MethodPost, "/api/hosts/bulk/enable", map[string][]string{"uuids": uuids}, nil)
}

// BulkDisableHosts disables the hosts identified by the given UUIDs.
func (c *Client) BulkDisableHosts(ctx context.Context, uuids []string) error {
	return c.doRequest(ctx, http.MethodPost, "/api/hosts/bulk/disable", map[string][]string{"uuids": uuids}, nil)
}

// BulkDeleteHosts deletes the hosts identified by the given UUIDs.
func (c *Client) BulkDeleteHosts(ctx context.Context, uuids []string) error {
	return c.doRequest(ctx, http.MethodPost, "/api/hosts/bulk/delete", map[string][]string{"uuids": uuids}, nil)
}

// ─── Bulk User Actions API ───

// bulkUserActionEndpoint maps a user bulk action string to its REST endpoint
// suffix under /api/users/bulk/.
var bulkUserActionEndpoint = map[string]string{
	"reset_traffic":       "reset-traffic",
	"revoke_subscription": "revoke-subscription",
	"delete":              "delete",
}

// BulkUserAction performs a bulk action (reset_traffic, revoke_subscription, or
// delete) on the given user UUIDs. All three endpoints accept a POST with a
// {"uuids": [...]} JSON body — the backend intentionally routes bulk delete via
// POST rather than the HTTP DELETE method.
func (c *Client) BulkUserAction(ctx context.Context, action string, uuids []string) error {
	suffix, ok := bulkUserActionEndpoint[action]
	if !ok {
		return fmt.Errorf("unknown user bulk action %q: must be one of reset_traffic, revoke_subscription, delete", action)
	}
	path := "/api/users/bulk/" + suffix
	return c.doRequest(ctx, http.MethodPost, path, map[string][]string{"uuids": uuids}, nil)
}

// BulkUserExtendExpiration extends the subscription expiration of the given
// users by the specified number of days via
// POST /api/users/bulk/extend-expiration-date with
// {"uuids": [...], "extendDays": N}.
func (c *Client) BulkUserExtendExpiration(ctx context.Context, uuids []string, extendDays int) error {
	body := map[string]any{
		"uuids":      uuids,
		"extendDays": extendDays,
	}
	return c.doRequest(ctx, http.MethodPost, "/api/users/bulk/extend-expiration-date", body, nil)
}

// ─── Bulk Node Actions API ───

// bulkNodeActionEnum maps a lowercase node bulk action string to the uppercase
// enum value expected by POST /api/nodes/bulk-actions.
var bulkNodeActionEnum = map[string]string{
	"enable":        "ENABLE",
	"disable":       "DISABLE",
	"restart":       "RESTART",
	"reset_traffic": "RESET_TRAFFIC",
}

// BulkNodeAction performs a bulk action (enable, disable, restart, or
// reset_traffic) on the given node UUIDs via POST /api/nodes/bulk-actions with
// {"uuids": [...], "action": "ENABLE"|"DISABLE"|"RESTART"|"RESET_TRAFFIC"}.
func (c *Client) BulkNodeAction(ctx context.Context, action string, uuids []string) error {
	enum, ok := bulkNodeActionEnum[action]
	if !ok {
		return fmt.Errorf("unknown node bulk action %q: must be one of enable, disable, restart, reset_traffic", action)
	}
	body := map[string]any{
		"uuids":  uuids,
		"action": enum,
	}
	return c.doRequest(ctx, http.MethodPost, "/api/nodes/bulk-actions", body, nil)
}

// ─── System API ───

func (c *Client) GetSystemHealth(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/health", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Config Profile API ───

func (c *Client) CreateConfigProfile(ctx context.Context, profile *ConfigProfile) (*ConfigProfile, error) {
	var out ConfigProfile
	if err := c.doRequest(ctx, http.MethodPost, "/api/config-profiles", profile, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetConfigProfileByUUID(ctx context.Context, uuid string) (*ConfigProfile, error) {
	var out ConfigProfile
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/config-profiles/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateConfigProfile(ctx context.Context, profile *ConfigProfile) (*ConfigProfile, error) {
	var out ConfigProfile
	if err := c.doRequest(ctx, http.MethodPatch, "/api/config-profiles", profile, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteConfigProfile(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/config-profiles/%s", uuid), nil, nil)
}

type configProfilesListResponse struct {
	Total          int             `json:"total"`
	ConfigProfiles []ConfigProfile `json:"configProfiles"`
}

func (c *Client) GetAllConfigProfiles(ctx context.Context) ([]ConfigProfile, error) {
	var out configProfilesListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/config-profiles", nil, &out); err != nil {
		return nil, err
	}
	return out.ConfigProfiles, nil
}

// ─── Subscription Settings API (singleton) ───

func (c *Client) GetSubscriptionSettings(ctx context.Context) (*SubscriptionSettings, error) {
	var out SubscriptionSettings
	if err := c.doRequest(ctx, http.MethodGet, "/api/subscription-settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSubscriptionSettings(ctx context.Context, settings *SubscriptionSettings) (*SubscriptionSettings, error) {
	var out SubscriptionSettings
	if err := c.doRequest(ctx, http.MethodPatch, "/api/subscription-settings", settings, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── Internal Squad API ───

func (c *Client) CreateInternalSquad(ctx context.Context, squad *InternalSquad) (*InternalSquad, error) {
	var out InternalSquad
	if err := c.doRequest(ctx, http.MethodPost, "/api/internal-squads", squad, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInternalSquadByUUID(ctx context.Context, uuid string) (*InternalSquad, error) {
	var out InternalSquad
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/internal-squads/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateInternalSquad(ctx context.Context, squad *InternalSquad) (*InternalSquad, error) {
	var out InternalSquad
	if err := c.doRequest(ctx, http.MethodPatch, "/api/internal-squads", squad, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteInternalSquad(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/internal-squads/%s", uuid), nil, nil)
}

func (c *Client) GetInternalSquadAccessibleNodes(ctx context.Context, uuid string) (*InternalSquadAccessibleNodes, error) {
	var out InternalSquadAccessibleNodes
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/internal-squads/%s/accessible-nodes", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── External Squad API ───

func (c *Client) CreateExternalSquad(ctx context.Context, squad *ExternalSquad) (*ExternalSquad, error) {
	var out ExternalSquad
	if err := c.doRequest(ctx, http.MethodPost, "/api/external-squads", squad, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetExternalSquadByUUID(ctx context.Context, uuid string) (*ExternalSquad, error) {
	var out ExternalSquad
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/external-squads/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateExternalSquad(ctx context.Context, squad *ExternalSquad) (*ExternalSquad, error) {
	var out ExternalSquad
	if err := c.doRequest(ctx, http.MethodPatch, "/api/external-squads", squad, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteExternalSquad(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/external-squads/%s", uuid), nil, nil)
}

// ─── Subscription Template API ───

func (c *Client) CreateSubscriptionTemplate(ctx context.Context, tmpl *SubscriptionTemplate) (*SubscriptionTemplate, error) {
	var out SubscriptionTemplate
	if err := c.doRequest(ctx, http.MethodPost, "/api/subscription-templates", tmpl, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSubscriptionTemplateByUUID(ctx context.Context, uuid string) (*SubscriptionTemplate, error) {
	var out SubscriptionTemplate
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscription-templates/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSubscriptionTemplate(ctx context.Context, tmpl *SubscriptionTemplate) (*SubscriptionTemplate, error) {
	var out SubscriptionTemplate
	if err := c.doRequest(ctx, http.MethodPatch, "/api/subscription-templates", tmpl, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSubscriptionTemplate(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/subscription-templates/%s", uuid), nil, nil)
}

// ─── Panel Settings API (singleton) ───

func (c *Client) GetPanelSettings(ctx context.Context) (*PanelSettings, error) {
	var out PanelSettings
	if err := c.doRequest(ctx, http.MethodGet, "/api/remnawave-settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePanelSettings(ctx context.Context, settings *PanelSettings) (*PanelSettings, error) {
	var out PanelSettings
	if err := c.doRequest(ctx, http.MethodPatch, "/api/remnawave-settings", settings, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── Snippet API ───

type snippetsListResponse struct {
	Total    int       `json:"total"`
	Snippets []Snippet `json:"snippets"`
}

func (c *Client) CreateSnippet(ctx context.Context, s *Snippet) (*snippetsListResponse, error) {
	var out snippetsListResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/snippets", s, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSnippets(ctx context.Context) (*snippetsListResponse, error) {
	var out snippetsListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/snippets", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSnippet(ctx context.Context, s *Snippet) (*snippetsListResponse, error) {
	var out snippetsListResponse
	if err := c.doRequest(ctx, http.MethodPatch, "/api/snippets", s, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSnippet(ctx context.Context, name string) error {
	return c.doRequest(ctx, http.MethodDelete, "/api/snippets", map[string]string{"name": name}, nil)
}

// ─── Node Plugin API ───

func (c *Client) CreateNodePlugin(ctx context.Context, p *NodePlugin) (*NodePlugin, error) {
	var out NodePlugin
	if err := c.doRequest(ctx, http.MethodPost, "/api/node-plugins", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetNodePluginByUUID(ctx context.Context, uuid string) (*NodePlugin, error) {
	var out NodePlugin
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/node-plugins/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateNodePlugin(ctx context.Context, p *NodePlugin) (*NodePlugin, error) {
	var out NodePlugin
	if err := c.doRequest(ctx, http.MethodPatch, "/api/node-plugins", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteNodePlugin(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/node-plugins/%s", uuid), nil, nil)
}

// ─── API Token API ───

func (c *Client) CreateApiToken(ctx context.Context, t *ApiToken) (*ApiToken, error) {
	if c.isVersion2_7(ctx) {
		return c.createApiTokenV27(ctx, t)
	}
	var out ApiToken
	if err := c.doRequest(ctx, http.MethodPost, "/api/tokens", t, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// createApiTokenV27 creates a token on Remnawave 2.7.x, which uses a
// different request field (tokenName instead of name, no expiresInDays
// or scopes) and a different response shape.
func (c *Client) createApiTokenV27(ctx context.Context, t *ApiToken) (*ApiToken, error) {
	payload := map[string]string{
		"tokenName": t.Name,
	}
	var resp struct {
		UUID      string `json:"uuid"`
		Token     string `json:"token"`
		TokenName string `json:"tokenName"`
	}
	if err := c.doRequest(ctx, http.MethodPost, "/api/tokens", payload, &resp); err != nil {
		return nil, err
	}
	return &ApiToken{
		UUID:  resp.UUID,
		Name:  resp.TokenName,
		Token: resp.Token,
		// 2.7.x does not return expireAt or scopes.
		Scopes: []string{"*"},
	}, nil
}

func (c *Client) DeleteApiToken(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/tokens/%s", uuid), nil, nil)
}

type apiTokensListResponse struct {
	Tokens []ApiToken `json:"tokens"`
}

// apiKeysListResponse is the 2.7.x variant: the array is named "apiKeys"
// and each item uses "tokenName" instead of "name", with no expireAt/scopes.
type apiKeysListResponse struct {
	APIKeys []apiKeyItem `json:"apiKeys"`
}

type apiKeyItem struct {
	UUID      string `json:"uuid"`
	TokenName string `json:"tokenName"`
}

func (c *Client) GetAllApiTokens(ctx context.Context) ([]ApiToken, error) {
	if c.isVersion2_7(ctx) {
		return c.getAllApiTokensV27(ctx)
	}
	var out apiTokensListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/tokens", nil, &out); err != nil {
		return nil, err
	}
	return out.Tokens, nil
}

func (c *Client) getAllApiTokensV27(ctx context.Context) ([]ApiToken, error) {
	var out apiKeysListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/tokens", nil, &out); err != nil {
		return nil, err
	}
	tokens := make([]ApiToken, len(out.APIKeys))
	for i, k := range out.APIKeys {
		tokens[i] = ApiToken{
			UUID:   k.UUID,
			Name:   k.TokenName,
			Scopes: []string{"*"},
		}
	}
	return tokens, nil
}

// ─── Passkey API ───

// passkeysResponse is the unwrapped {passkeys: [...]} payload after decodeResponse
// strips the outer {response: ...} envelope.
type passkeysResponse struct {
	Passkeys []Passkey `json:"passkeys"`
}

func (c *Client) GetAllPasskeys(ctx context.Context) ([]Passkey, error) {
	var out passkeysResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/passkeys", nil, &out); err != nil {
		return nil, err
	}
	return out.Passkeys, nil
}

func (c *Client) DeletePasskey(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/passkeys/%s", uuid), nil, nil)
}

func (c *Client) GetSystemStats(ctx context.Context, tz string) (map[string]any, error) {
	path := "/api/system/stats"
	if tz != "" {
		path += "?" + url.Values{"tz": {tz}}.Encode()
	}
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSystemRecap(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/stats/recap", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetNodesMetrics(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/nodes/metrics", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Keygen API ───

func (c *Client) GetKeygenPubKey(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/keygen", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Infra Billing API ───

func (c *Client) CreateInfraProvider(ctx context.Context, p *InfraProvider) (*InfraProvider, error) {
	var out InfraProvider
	if err := c.doRequest(ctx, http.MethodPost, "/api/infra-billing/providers", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInfraProviderByUUID(ctx context.Context, uuid string) (*InfraProvider, error) {
	var out InfraProvider
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/infra-billing/providers/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateInfraProvider(ctx context.Context, p *InfraProvider) (*InfraProvider, error) {
	var out InfraProvider
	if err := c.doRequest(ctx, http.MethodPatch, "/api/infra-billing/providers", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteInfraProvider(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/infra-billing/providers/%s", uuid), nil, nil)
}

// ─── Infra Billing Node API ───

type billingNodesResponse struct {
	TotalBillingNodes          int              `json:"totalBillingNodes"`
	BillingNodes               []BillingNode    `json:"billingNodes"`
	AvailableBillingNodes      []map[string]any `json:"availableBillingNodes"`
	TotalAvailableBillingNodes int              `json:"totalAvailableBillingNodes"`
	Stats                      map[string]any   `json:"stats"`
}

func (c *Client) CreateBillingNode(ctx context.Context, req map[string]any) (*billingNodesResponse, error) {
	var out billingNodesResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/infra-billing/nodes", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateBillingNode(ctx context.Context, req map[string]any) (*billingNodesResponse, error) {
	var out billingNodesResponse
	if err := c.doRequest(ctx, http.MethodPatch, "/api/infra-billing/nodes", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBillingNodes(ctx context.Context) (*billingNodesResponse, error) {
	var out billingNodesResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/infra-billing/nodes", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBillingNode(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/infra-billing/nodes/%s", uuid), nil, nil)
}

// ─── Infra Billing History API ───

type billingHistoryResponse struct {
	Records []BillingHistoryRecord `json:"records"`
	Total   int                    `json:"total"`
}

func (c *Client) CreateBillingHistory(ctx context.Context, req map[string]any) (*billingHistoryResponse, error) {
	var out billingHistoryResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/infra-billing/history", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBillingHistory(ctx context.Context) (*billingHistoryResponse, error) {
	var out billingHistoryResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/infra-billing/history", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBillingHistory(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/infra-billing/history/%s", uuid), nil, nil)
}

// ─── Subscriptions API ───

func (c *Client) GetSubscriptionByUUID(ctx context.Context, uuid string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscriptions/by-uuid/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSubscriptionByUsername(ctx context.Context, username string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscriptions/by-username/%s", username), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSubscriptionByShortUUID(ctx context.Context, shortUUID string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscriptions/by-short-uuid/%s", shortUUID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Subscription Request History API ───

func (c *Client) GetSubscriptionRequestHistory(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/subscription-request-history?size=1000", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Bandwidth Stats API ───

func (c *Client) GetBandwidthStatsNodes(ctx context.Context, start, end string, topNodesLimit int) (map[string]any, error) {
	query := url.Values{"start": {start}, "end": {end}}
	if topNodesLimit > 0 {
		query.Set("topNodesLimit", strconv.Itoa(topNodesLimit))
	}
	path := "/api/bandwidth-stats/nodes?" + query.Encode()
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetBandwidthStatsUser(ctx context.Context, uuid, start, end string, topNodesLimit int) (map[string]any, error) {
	query := url.Values{"start": {start}, "end": {end}}
	if topNodesLimit > 0 {
		query.Set("topNodesLimit", strconv.Itoa(topNodesLimit))
	}
	path := fmt.Sprintf("/api/bandwidth-stats/users/%s?%s", uuid, query.Encode())
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── SubpageConfig API ───

func (c *Client) CreateSubpageConfig(ctx context.Context, sc *SubpageConfig) (*SubpageConfig, error) {
	var out SubpageConfig
	if err := c.doRequest(ctx, http.MethodPost, "/api/subscription-page-configs", sc, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSubpageConfigByUUID(ctx context.Context, uuid string) (*SubpageConfig, error) {
	var out SubpageConfig
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscription-page-configs/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSubpageConfig(ctx context.Context, sc *SubpageConfig) (*SubpageConfig, error) {
	var out SubpageConfig
	if err := c.doRequest(ctx, http.MethodPatch, "/api/subscription-page-configs", sc, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSubpageConfig(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/subscription-page-configs/%s", uuid), nil, nil)
}

// ─── Metadata API ───

func (c *Client) GetUserMetadata(ctx context.Context, uuid string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/metadata/user/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpsertUserMetadata(ctx context.Context, uuid string, metadata map[string]any) (map[string]any, error) {
	body := map[string]any{"metadata": metadata}
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/metadata/user/%s", uuid), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetNodeMetadata(ctx context.Context, uuid string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/metadata/node/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpsertNodeMetadata(ctx context.Context, uuid string, metadata map[string]any) (map[string]any, error) {
	body := map[string]any{"metadata": metadata}
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/metadata/node/%s", uuid), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Misc Stats API ───

func (c *Client) GetBandwidthRealtime(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/bandwidth-stats/nodes/realtime", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSystemBandwidthStats(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/stats/bandwidth", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSystemNodesStats(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/system/stats/nodes", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSubscriptionRequestHistoryStats(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/subscription-request-history/stats", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetConnectionKeys(ctx context.Context, uuid string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/subscriptions/connection-keys/%s", uuid), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── HWID API ───

func (c *Client) CreateHwidDevice(ctx context.Context, req map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodPost, "/api/hwid/devices", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteHwidDevice(ctx context.Context, req map[string]any) error {
	return c.doRequest(ctx, http.MethodPost, "/api/hwid/devices/delete", req, nil)
}

func (c *Client) GetUserHwidDevices(ctx context.Context, userUuid string) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/hwid/devices/%s?size=1000", userUuid), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetHwidStats(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/hwid/devices/stats", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetHwidTopUsers(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doRequest(ctx, http.MethodGet, "/api/hwid/devices/top-users", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── IP Control API ───

// ipControlJobResponse is the response from POST /api/ip-control/fetch-ips/:uuid.
type ipControlJobResponse struct {
	JobID string `json:"jobId"`
}

// ipControlJobResult is the response from GET /api/ip-control/fetch-ips/result/:jobId.
type ipControlJobResult struct {
	IsCompleted bool                   `json:"isCompleted"`
	IsFailed    bool                   `json:"isFailed"`
	Progress    ipControlJobProgress   `json:"progress"`
	Result      *ipControlJobResultDat `json:"result"`
}

type ipControlJobProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Percent   int `json:"percent"`
}

// ipControlJobResultDat is the nullable result payload.
type ipControlJobResultDat struct {
	Success  bool            `json:"success"`
	UserUUID string          `json:"userUuid"`
	UserID   string          `json:"userId"`
	Nodes    []ipControlNode `json:"nodes"`
}

type ipControlNode struct {
	NodeUUID    string        `json:"nodeUuid"`
	NodeName    string        `json:"nodeName"`
	CountryCode string        `json:"countryCode"`
	IPs         []ipControlIP `json:"ips"`
}

type ipControlIP struct {
	IP       string `json:"ip"`
	LastSeen string `json:"lastSeen"`
}

// FetchUserIPs initiates an async job to fetch the IPs a user is connected
// from, then polls for the result until it completes or the context is
// cancelled. It returns the list of IPs.
func (c *Client) FetchUserIPs(ctx context.Context, userUUID string) ([]string, error) {
	// 1. Start the job
	var jobResp ipControlJobResponse
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/ip-control/fetch-ips/%s", userUUID), nil, &jobResp); err != nil {
		return nil, fmt.Errorf("failed to start fetch-ips job: %w", err)
	}
	if jobResp.JobID == "" {
		return nil, errors.New("fetch-ips job started but no jobId returned")
	}

	// 2. Poll for the result
	const (
		pollInterval = 2 * time.Second
		pollTimeout  = 120 * time.Second
	)
	deadline := time.Now().Add(pollTimeout)
	for {
		var result ipControlJobResult
		if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/ip-control/fetch-ips/result/%s", jobResp.JobID), nil, &result); err != nil {
			return nil, fmt.Errorf("failed to fetch-ips result for jobId %s: %w", jobResp.JobID, err)
		}

		if result.IsFailed {
			return nil, fmt.Errorf("fetch-ips job %s failed", jobResp.JobID)
		}
		if result.IsCompleted && result.Result != nil {
			// Flatten all IPs across all nodes.
			var ips []string
			for _, node := range result.Result.Nodes {
				for _, ip := range node.IPs {
					ips = append(ips, ip.IP)
				}
			}
			return ips, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for fetch-ips job %s after %s", jobResp.JobID, pollTimeout)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// DropConnections drops all active connections for the given user UUID.
//
// Deprecated: use DropConnectionsV2 for the full API schema (drop by IP, target nodes).
func (c *Client) DropConnections(ctx context.Context, userUUID string) error {
	body := map[string]string{"userUuid": userUUID}
	return c.doRequest(ctx, http.MethodPost, "/api/ip-control/drop-connections", body, nil)
}

// DropConnectionsV2 drops connections using the full IP Control API schema.
// body must match { dropBy: { by: "userUuids"|"ipAddresses", ... }, targetNodes: { target: "allNodes"|"specificNodes", ... } }.
func (c *Client) DropConnectionsV2(ctx context.Context, body map[string]any) (bool, error) {
	var out struct {
		EventSent bool `json:"eventSent"`
	}
	if err := c.doRequest(ctx, http.MethodPost, "/api/ip-control/drop-connections", body, &out); err != nil {
		return false, err
	}
	return out.EventSent, nil
}
