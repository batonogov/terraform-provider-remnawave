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

	proxyHeaders bool
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

	c.authMu.Lock()
	c.accessToken = resp.AccessToken
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
	c.setProxyHeaders(req)

	// On 401, try re-authenticating once (unless using static API token).
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized && c.apiToken == "" {
		// #nosec G104 -- discarding body before re-auth; not actionable
		_ = resp.Body.Close()
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
		c.setProxyHeaders(req2)
		// Reset body reader for retry
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return err
			}
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
	base.Path = strings.TrimSuffix(base.Path, "/") + path
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

type usersListResponse struct {
	Total int    `json:"total"`
	Users []User `json:"users"`
}

func (c *Client) GetAllUsers(ctx context.Context) ([]User, error) {
	var out usersListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/users", nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
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
	var out ApiToken
	if err := c.doRequest(ctx, http.MethodPost, "/api/tokens", t, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteApiToken(ctx context.Context, uuid string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/tokens/%s", uuid), nil, nil)
}

type apiTokensListResponse struct {
	Tokens []ApiToken `json:"tokens"`
}

func (c *Client) GetAllApiTokens(ctx context.Context) ([]ApiToken, error) {
	var out apiTokensListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/tokens", nil, &out); err != nil {
		return nil, err
	}
	return out.Tokens, nil
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
	if err := c.doRequest(ctx, http.MethodGet, "/api/subscription-request-history", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
