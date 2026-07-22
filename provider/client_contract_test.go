package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type clientContractCase struct {
	name         string
	method       string
	path         string
	query        map[string]string
	args         []any
	wantJSON     map[string]any
	noBody       bool
	mockResponse string
}

// TestClientAPIContracts exercises every exported API operation against a
// local server. It protects the method/path/query contract without requiring a
// Docker panel and ensures every thin wrapper still goes through the shared
// authentication and response-decoding code.
func TestClientAPIContracts(t *testing.T) {
	tests := []clientContractCase{
		{name: "CreateUser", method: http.MethodPost, path: "/api/users", args: []any{&User{Username: "alice", ExpireAt: "2027-01-01T00:00:00Z"}}},
		{name: "GetUserByUUID", method: http.MethodGet, path: "/api/users/item-id", args: []any{"item-id"}},
		{name: "UpdateUser", method: http.MethodPatch, path: "/api/users", args: []any{&User{UUID: "item-id", Username: "alice", ExpireAt: "2027-01-01T00:00:00Z"}}},
		{name: "DeleteUser", method: http.MethodDelete, path: "/api/users/item-id", args: []any{"item-id"}},
		{name: "GetAllUsers", method: http.MethodGet, path: "/api/users", query: map[string]string{"size": "1000"}},

		{name: "CreateNode", method: http.MethodPost, path: "/api/nodes", args: []any{&Node{Name: "node", Address: "127.0.0.1"}}},
		{name: "GetAllNodes", method: http.MethodGet, path: "/api/nodes"},
		{name: "GetNodeByUUID", method: http.MethodGet, path: "/api/nodes/item-id", args: []any{"item-id"}},
		{name: "UpdateNode", method: http.MethodPatch, path: "/api/nodes", args: []any{&Node{UUID: "item-id", Name: "node", Address: "127.0.0.1"}}},
		{name: "DeleteNode", method: http.MethodDelete, path: "/api/nodes/item-id", args: []any{"item-id"}},
		{name: "EnableNode", method: http.MethodPost, path: "/api/nodes/item-id/actions/enable", args: []any{"item-id"}, noBody: true},
		{name: "DisableNode", method: http.MethodPost, path: "/api/nodes/item-id/actions/disable", args: []any{"item-id"}, noBody: true},
		{name: "RestartNode", method: http.MethodPost, path: "/api/nodes/item-id/actions/restart", args: []any{"item-id", true}, wantJSON: map[string]any{"forceRestart": true}},
		{name: "ResetNodeTraffic", method: http.MethodPost, path: "/api/nodes/item-id/actions/reset-traffic", args: []any{"item-id"}, noBody: true},

		{name: "CreateHost", method: http.MethodPost, path: "/api/hosts", args: []any{&Host{Remark: "host", Address: "host.example.com", Port: 443}}},
		{name: "GetAllHosts", method: http.MethodGet, path: "/api/hosts"},
		{name: "GetHostByUUID", method: http.MethodGet, path: "/api/hosts/item-id", args: []any{"item-id"}},
		{name: "UpdateHost", method: http.MethodPatch, path: "/api/hosts", args: []any{&Host{UUID: "item-id", Remark: "host", Address: "host.example.com", Port: 443}}},
		{name: "DeleteHost", method: http.MethodDelete, path: "/api/hosts/item-id", args: []any{"item-id"}},
		{name: "GetHostTags", method: http.MethodGet, path: "/api/hosts/tags"},
		{name: "BulkEnableHosts", method: http.MethodPost, path: "/api/hosts/bulk/enable", args: []any{[]string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkDisableHosts", method: http.MethodPost, path: "/api/hosts/bulk/disable", args: []any{[]string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkDeleteHosts", method: http.MethodPost, path: "/api/hosts/bulk/delete", args: []any{[]string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkUserActionResetTraffic", method: http.MethodPost, path: "/api/users/bulk/reset-traffic", args: []any{"reset_traffic", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkUserActionRevoke", method: http.MethodPost, path: "/api/users/bulk/revoke-subscription", args: []any{"revoke_subscription", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkUserActionDelete", method: http.MethodPost, path: "/api/users/bulk/delete", args: []any{"delete", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}}},
		{name: "BulkUserExtendExpiration", method: http.MethodPost, path: "/api/users/bulk/extend-expiration-date", args: []any{[]string{"item-id"}, 7}, wantJSON: map[string]any{"uuids": []any{"item-id"}, "extendDays": float64(7)}},
		{name: "BulkNodeActionEnable", method: http.MethodPost, path: "/api/nodes/bulk-actions", args: []any{"enable", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}, "action": "ENABLE"}},
		{name: "BulkNodeActionDisable", method: http.MethodPost, path: "/api/nodes/bulk-actions", args: []any{"disable", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}, "action": "DISABLE"}},
		{name: "BulkNodeActionRestart", method: http.MethodPost, path: "/api/nodes/bulk-actions", args: []any{"restart", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}, "action": "RESTART"}},
		{name: "BulkNodeActionResetTraffic", method: http.MethodPost, path: "/api/nodes/bulk-actions", args: []any{"reset_traffic", []string{"item-id"}}, wantJSON: map[string]any{"uuids": []any{"item-id"}, "action": "RESET_TRAFFIC"}},
		{name: "GetSystemHealth", method: http.MethodGet, path: "/api/system/health"},

		{name: "CreateConfigProfile", method: http.MethodPost, path: "/api/config-profiles", args: []any{&ConfigProfile{Name: "profile"}}},
		{name: "GetConfigProfileByUUID", method: http.MethodGet, path: "/api/config-profiles/item-id", args: []any{"item-id"}},
		{name: "UpdateConfigProfile", method: http.MethodPatch, path: "/api/config-profiles", args: []any{&ConfigProfile{UUID: "item-id", Name: "profile"}}},
		{name: "DeleteConfigProfile", method: http.MethodDelete, path: "/api/config-profiles/item-id", args: []any{"item-id"}},
		{name: "GetAllConfigProfiles", method: http.MethodGet, path: "/api/config-profiles"},

		{name: "GetSubscriptionSettings", method: http.MethodGet, path: "/api/subscription-settings"},
		{name: "UpdateSubscriptionSettings", method: http.MethodPatch, path: "/api/subscription-settings", args: []any{&SubscriptionSettings{UUID: "item-id"}}},

		{name: "CreateInternalSquad", method: http.MethodPost, path: "/api/internal-squads", args: []any{&InternalSquad{Name: "squad", Inbounds: []string{}}}},
		{name: "GetInternalSquadByUUID", method: http.MethodGet, path: "/api/internal-squads/item-id", args: []any{"item-id"}},
		{name: "UpdateInternalSquad", method: http.MethodPatch, path: "/api/internal-squads", args: []any{&InternalSquad{UUID: "item-id", Name: "squad", Inbounds: []string{}}}},
		{name: "DeleteInternalSquad", method: http.MethodDelete, path: "/api/internal-squads/item-id", args: []any{"item-id"}},
		{name: "GetInternalSquadAccessibleNodes", method: http.MethodGet, path: "/api/internal-squads/item-id/accessible-nodes", args: []any{"item-id"}},

		{name: "CreateExternalSquad", method: http.MethodPost, path: "/api/external-squads", args: []any{&ExternalSquad{Name: "squad"}}},
		{name: "GetExternalSquadByUUID", method: http.MethodGet, path: "/api/external-squads/item-id", args: []any{"item-id"}},
		{name: "UpdateExternalSquad", method: http.MethodPatch, path: "/api/external-squads", args: []any{&ExternalSquad{UUID: "item-id", Name: "squad"}}},
		{name: "DeleteExternalSquad", method: http.MethodDelete, path: "/api/external-squads/item-id", args: []any{"item-id"}},

		{name: "CreateSubscriptionTemplate", method: http.MethodPost, path: "/api/subscription-templates", args: []any{&SubscriptionTemplate{Name: "template"}}},
		{name: "GetSubscriptionTemplateByUUID", method: http.MethodGet, path: "/api/subscription-templates/item-id", args: []any{"item-id"}},
		{name: "UpdateSubscriptionTemplate", method: http.MethodPatch, path: "/api/subscription-templates", args: []any{&SubscriptionTemplate{UUID: "item-id", Name: "template"}}},
		{name: "DeleteSubscriptionTemplate", method: http.MethodDelete, path: "/api/subscription-templates/item-id", args: []any{"item-id"}},

		{name: "GetPanelSettings", method: http.MethodGet, path: "/api/remnawave-settings"},
		{name: "UpdatePanelSettings", method: http.MethodPatch, path: "/api/remnawave-settings", args: []any{&PanelSettings{}}},

		{name: "CreateSnippet", method: http.MethodPost, path: "/api/snippets", args: []any{&Snippet{Name: "snippet"}}},
		{name: "GetSnippets", method: http.MethodGet, path: "/api/snippets"},
		{name: "UpdateSnippet", method: http.MethodPatch, path: "/api/snippets", args: []any{&Snippet{Name: "snippet"}}},
		{name: "DeleteSnippet", method: http.MethodDelete, path: "/api/snippets", args: []any{"snippet"}, wantJSON: map[string]any{"name": "snippet"}},

		{name: "CreateNodePlugin", method: http.MethodPost, path: "/api/node-plugins", args: []any{&NodePlugin{Name: "plugin"}}},
		{name: "GetNodePluginByUUID", method: http.MethodGet, path: "/api/node-plugins/item-id", args: []any{"item-id"}},
		{name: "UpdateNodePlugin", method: http.MethodPatch, path: "/api/node-plugins", args: []any{&NodePlugin{UUID: "item-id", Name: "plugin"}}},
		{name: "DeleteNodePlugin", method: http.MethodDelete, path: "/api/node-plugins/item-id", args: []any{"item-id"}},

		{name: "CreateApiToken", method: http.MethodPost, path: "/api/tokens", args: []any{&ApiToken{Name: "token", ExpiresInDays: 7, Scopes: []string{"users:read"}}}, wantJSON: map[string]any{"name": "token", "expiresInDays": float64(7), "scopes": []any{"users:read"}}},
		{name: "DeleteApiToken", method: http.MethodDelete, path: "/api/tokens/item-id", args: []any{"item-id"}},
		{name: "GetAllApiTokens", method: http.MethodGet, path: "/api/tokens"},

		{name: "GetSystemStats", method: http.MethodGet, path: "/api/system/stats", args: []any{"Europe/Moscow"}, query: map[string]string{"tz": "Europe/Moscow"}},
		{name: "GetSystemStatsWithoutTimezone", method: http.MethodGet, path: "/api/system/stats", args: []any{""}},
		{name: "GetSystemRecap", method: http.MethodGet, path: "/api/system/stats/recap"},
		{name: "GetNodesMetrics", method: http.MethodGet, path: "/api/system/nodes/metrics"},
		{name: "GetKeygenPubKey", method: http.MethodGet, path: "/api/keygen"},

		{name: "CreateInfraProvider", method: http.MethodPost, path: "/api/infra-billing/providers", args: []any{&InfraProvider{Name: "provider"}}},
		{name: "GetInfraProviderByUUID", method: http.MethodGet, path: "/api/infra-billing/providers/item-id", args: []any{"item-id"}},
		{name: "UpdateInfraProvider", method: http.MethodPatch, path: "/api/infra-billing/providers", args: []any{&InfraProvider{UUID: "item-id", Name: "provider"}}},
		{name: "DeleteInfraProvider", method: http.MethodDelete, path: "/api/infra-billing/providers/item-id", args: []any{"item-id"}},

		{name: "CreateBillingNode", method: http.MethodPost, path: "/api/infra-billing/nodes", args: []any{map[string]any{"name": "node"}}, wantJSON: map[string]any{"name": "node"}},
		{name: "UpdateBillingNode", method: http.MethodPatch, path: "/api/infra-billing/nodes", args: []any{map[string]any{"uuid": "item-id"}}, wantJSON: map[string]any{"uuid": "item-id"}},
		{name: "GetBillingNodes", method: http.MethodGet, path: "/api/infra-billing/nodes"},
		{name: "DeleteBillingNode", method: http.MethodDelete, path: "/api/infra-billing/nodes/item-id", args: []any{"item-id"}},

		{name: "CreateBillingHistory", method: http.MethodPost, path: "/api/infra-billing/history", args: []any{map[string]any{"amount": float64(42)}}, wantJSON: map[string]any{"amount": float64(42)}},
		{name: "GetBillingHistory", method: http.MethodGet, path: "/api/infra-billing/history"},
		{name: "GetBillingHistoryPaged", method: http.MethodGet, path: "/api/infra-billing/history", query: map[string]string{"size": "500", "start": "0"}, args: []any{0, 500}},
		{name: "DeleteBillingHistory", method: http.MethodDelete, path: "/api/infra-billing/history/item-id", args: []any{"item-id"}},

		{name: "GetSubscriptionByUUID", method: http.MethodGet, path: "/api/subscriptions/by-uuid/item-id", args: []any{"item-id"}},
		{name: "GetSubscriptionByUsername", method: http.MethodGet, path: "/api/subscriptions/by-username/alice", args: []any{"alice"}},
		{name: "GetSubscriptionByShortUUID", method: http.MethodGet, path: "/api/subscriptions/by-short-uuid/short-id", args: []any{"short-id"}},
		{name: "GetSubscriptionRequestHistory", method: http.MethodGet, path: "/api/subscription-request-history", query: map[string]string{"size": "1000"}},

		{name: "GetBandwidthStatsNodes", method: http.MethodGet, path: "/api/bandwidth-stats/nodes", args: []any{"2026-01-01T00:00:00+03:00", "2026-01-02T00:00:00+03:00", 5}, query: map[string]string{"start": "2026-01-01T00:00:00+03:00", "end": "2026-01-02T00:00:00+03:00", "topNodesLimit": "5"}},
		{name: "GetBandwidthStatsNodesWithoutLimit", method: http.MethodGet, path: "/api/bandwidth-stats/nodes", args: []any{"start", "end", 0}, query: map[string]string{"start": "start", "end": "end"}},
		{name: "GetBandwidthStatsUser", method: http.MethodGet, path: "/api/bandwidth-stats/users/item-id", args: []any{"item-id", "start", "end", 3}, query: map[string]string{"start": "start", "end": "end", "topNodesLimit": "3"}},

		{name: "CreateSubpageConfig", method: http.MethodPost, path: "/api/subscription-page-configs", args: []any{&SubpageConfig{Name: "config"}}},
		{name: "GetSubpageConfigByUUID", method: http.MethodGet, path: "/api/subscription-page-configs/item-id", args: []any{"item-id"}},
		{name: "UpdateSubpageConfig", method: http.MethodPatch, path: "/api/subscription-page-configs", args: []any{&SubpageConfig{UUID: "item-id", Name: "config"}}},
		{name: "DeleteSubpageConfig", method: http.MethodDelete, path: "/api/subscription-page-configs/item-id", args: []any{"item-id"}},

		{name: "GetUserMetadata", method: http.MethodGet, path: "/api/metadata/user/item-id", args: []any{"item-id"}},
		{name: "UpsertUserMetadata", method: http.MethodPut, path: "/api/metadata/user/item-id", args: []any{"item-id", map[string]any{"key": "value"}}, wantJSON: map[string]any{"metadata": map[string]any{"key": "value"}}},
		{name: "GetNodeMetadata", method: http.MethodGet, path: "/api/metadata/node/item-id", args: []any{"item-id"}},
		{name: "UpsertNodeMetadata", method: http.MethodPut, path: "/api/metadata/node/item-id", args: []any{"item-id", map[string]any{"key": "value"}}, wantJSON: map[string]any{"metadata": map[string]any{"key": "value"}}},

		{name: "GetBandwidthRealtime", method: http.MethodGet, path: "/api/bandwidth-stats/nodes/realtime"},
		{name: "GetSystemBandwidthStats", method: http.MethodGet, path: "/api/system/stats/bandwidth"},
		{name: "GetSystemNodesStats", method: http.MethodGet, path: "/api/system/stats/nodes"},
		{name: "GetSubscriptionRequestHistoryStats", method: http.MethodGet, path: "/api/subscription-request-history/stats"},
		{name: "GetConnectionKeys", method: http.MethodGet, path: "/api/subscriptions/connection-keys/item-id", args: []any{"item-id"}},

		{name: "CreateHwidDevice", method: http.MethodPost, path: "/api/hwid/devices", args: []any{map[string]any{"hwid": "device-id"}}, wantJSON: map[string]any{"hwid": "device-id"}},
		{name: "DeleteHwidDevice", method: http.MethodPost, path: "/api/hwid/devices/delete", args: []any{map[string]any{"hwid": "device-id"}}, wantJSON: map[string]any{"hwid": "device-id"}},
		{name: "GetUserHwidDevices", method: http.MethodGet, path: "/api/hwid/devices/item-id", args: []any{"item-id"}, query: map[string]string{"size": "1000"}},
		{name: "GetHwidStats", method: http.MethodGet, path: "/api/hwid/devices/stats"},
		{name: "GetHwidTopUsers", method: http.MethodGet, path: "/api/hwid/devices/top-users"},
		{name: "UserAction", method: http.MethodPost, path: "/api/users/item-id/actions/reset-traffic", args: []any{"item-id", "reset_traffic"}, noBody: true},
		{name: "FetchUserIPs", method: http.MethodPost, path: "/api/ip-control/fetch-ips/item-id", args: []any{"item-id"}, noBody: true, mockResponse: `{"response":{"jobId":"job-1"}}`},
		{name: "DropConnections", method: http.MethodPost, path: "/api/ip-control/drop-connections", args: []any{"item-id"}, wantJSON: map[string]any{"userUuid": "item-id"}},
		{name: "DropConnectionsV2", method: http.MethodPost, path: "/api/ip-control/drop-connections", args: []any{map[string]any{"dropBy": map[string]any{"by": "userUuids", "userUuids": []any{"item-id"}}, "targetNodes": map[string]any{"target": "allNodes"}}}, wantJSON: map[string]any{"dropBy": map[string]any{"by": "userUuids", "userUuids": []any{"item-id"}}, "targetNodes": map[string]any{"target": "allNodes"}}},
		{name: "GetAllPasskeys", method: http.MethodGet, path: "/api/passkeys"},
		{name: "DeletePasskey", method: http.MethodDelete, path: "/api/passkeys", args: []any{"item-id"}, wantJSON: map[string]any{"id": "item-id"}},
	}
	coveredMethods := make(map[string]struct{}, len(tests))
	for _, tt := range tests {
		coveredMethods[clientContractMethodName(tt.name)] = struct{}{}
	}
	clientType := reflect.TypeOf((*Client)(nil))
	for i := 0; i < clientType.NumMethod(); i++ {
		methodName := clientType.Method(i).Name
		if _, ok := coveredMethods[methodName]; !ok {
			t.Errorf("exported Client method %s has no API contract test", methodName)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Version detection probe — respond and skip contract assertion.
				if r.URL.Path == "/api/system/metadata" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"response":{"version":"2.8.0"}}`)
					return
				}
				// For async methods that poll, serve the poll response without assertions.
				if tt.mockResponse != "" && r.URL.Path != tt.path {
					_, _ = io.WriteString(w, `{"response":{"isCompleted":true,"isFailed":false,"progress":{"total":1,"completed":1,"percent":100},"result":{"success":true,"userUuid":"item-id","userId":"1","nodes":[{"nodeUuid":"node-1","nodeName":"test","countryCode":"US","ips":[]}]}}}`)
					return
				}
				if r.Method != tt.method {
					t.Errorf("method = %s, want %s", r.Method, tt.method)
				}
				if r.URL.Path != tt.path {
					t.Errorf("path = %q, want %q", r.URL.Path, tt.path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer contract-token" {
					t.Errorf("Authorization = %q", got)
				}
				if got := r.Header.Get("X-Remnawave-Client-Type"); got != "" {
					t.Errorf("static API token request has X-Remnawave-Client-Type = %q", got)
				}
				if len(r.URL.Query()) != len(tt.query) {
					t.Errorf("query = %#v, want %#v", r.URL.Query(), tt.query)
				}
				for key, want := range tt.query {
					if got := r.URL.Query().Get(key); got != want {
						t.Errorf("query %q = %q, want %q", key, got, want)
					}
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read body: %v", err)
				}
				wantBody := !tt.noBody && (tt.method == http.MethodPost || tt.method == http.MethodPatch || tt.method == http.MethodPut || tt.wantJSON != nil)
				if wantBody && len(body) == 0 {
					t.Errorf("request body is empty")
				}
				if !wantBody && len(body) != 0 {
					t.Errorf("unexpected request body: %s", body)
				}
				if tt.wantJSON != nil {
					var got map[string]any
					if err := json.Unmarshal(body, &got); err != nil {
						t.Errorf("decode request JSON: %v", err)
					} else if !reflect.DeepEqual(got, tt.wantJSON) {
						t.Errorf("request JSON = %#v, want %#v", got, tt.wantJSON)
					}
				}

				w.Header().Set("Content-Type", "application/json")
				response := `{}`
				if tt.name == "GetAllNodes" || tt.name == "GetAllHosts" {
					response = `[]`
				}
				if tt.name == "GetHostTags" {
					response = `{"tags":[]}`
				}
				if tt.name == "GetAllPasskeys" {
					response = `{"passkeys":[]}`
				}
				if tt.mockResponse != "" {
					// For async methods that poll (FetchUserIPs), serve mockResponse
					// for the initial request and a completed result for the poll.
					if r.URL.Path != tt.path {
						_, _ = io.WriteString(w, `{"response":{"isCompleted":true,"isFailed":false,"progress":{"total":1,"completed":1,"percent":100},"result":{"success":true,"userUuid":"item-id","userId":"1","nodes":[{"nodeUuid":"node-1","nodeName":"test","countryCode":"US","ips":[]}]}}}`)
					} else {
						_, _ = io.WriteString(w, tt.mockResponse)
					}
				} else {
					_, _ = io.WriteString(w, `{"response":`+response+`}`)
				}
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "contract-token"})
			if err != nil {
				t.Fatal(err)
			}
			method := reflect.ValueOf(client).MethodByName(clientContractMethodName(tt.name))
			if !method.IsValid() {
				t.Fatalf("client method %s does not exist", tt.name)
			}
			args := []reflect.Value{reflect.ValueOf(context.Background())}
			for _, arg := range tt.args {
				args = append(args, reflect.ValueOf(arg))
			}
			results := method.Call(args)
			if len(results) == 0 {
				t.Fatalf("method %s returned no values", tt.name)
			}
			errValue := results[len(results)-1]
			if !errValue.IsNil() {
				t.Fatalf("%s() error = %v", tt.name, errValue.Interface())
			}
		})
	}
}

func clientContractMethodName(caseName string) string {
	switch caseName {
	case "GetSystemStatsWithoutTimezone":
		return "GetSystemStats"
	case "GetBandwidthStatsNodesWithoutLimit":
		return "GetBandwidthStatsNodes"
	case "BulkUserActionResetTraffic", "BulkUserActionRevoke", "BulkUserActionDelete":
		return "BulkUserAction"
	case "BulkNodeActionEnable", "BulkNodeActionDisable", "BulkNodeActionRestart", "BulkNodeActionResetTraffic":
		return "BulkNodeAction"
	default:
		return caseName
	}
}

// captureRequestServer returns a test server that records every request body
// it receives on the given path/method, so callers can inspect the exact JSON
// field names the provider serializes.
func captureRequestServer(t *testing.T, method, path string, captured *[]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/metadata" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"version":"2.8.0"}}`)
			return
		}
		if r.Method != method {
			t.Errorf("method = %s, want %s", r.Method, method)
		}
		if r.URL.Path != path {
			t.Errorf("path = %q, want %q", r.URL.Path, path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		*captured = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{}}`)
	}))
}

// TestNodeCreateContract verifies the JSON field names sent to POST /api/nodes.
// Drift in any of: name, address, port, configProfile.activeConfigProfileUuid,
// configProfile.activeInbounds[] would break node creation.
func TestNodeCreateContract(t *testing.T) {
	var captured []byte
	server := captureRequestServer(t, http.MethodPost, "/api/nodes", &captured)
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "contract-token"})
	if err != nil {
		t.Fatal(err)
	}

	port := 443
	_, err = client.CreateNode(context.Background(), &Node{
		Name:    "test-node",
		Address: "10.0.0.1",
		Port:    &port,
		ConfigProfile: &NodeConfigProfile{
			ActiveConfigProfileUUID: "profile-uuid-123",
			ActiveInbounds:          []NodeConfigProfileInbound{{UUID: "inbound-uuid-1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(captured, &got); err != nil {
		t.Fatalf("decode request JSON: %v\nbody: %s", err, captured)
	}

	// Verify top-level fields the API requires.
	for _, k := range []string{"name", "address", "port", "configProfile"} {
		if _, ok := got[k]; !ok {
			t.Errorf("expected top-level JSON key %q in node create body, got: %s", k, captured)
		}
	}

	// Verify nested configProfile fields.
	cp, ok := got["configProfile"].(map[string]any)
	if !ok {
		t.Fatalf("configProfile is not an object, got: %s", captured)
	}
	for _, k := range []string{"activeConfigProfileUuid", "activeInbounds"} {
		if _, ok := cp[k]; !ok {
			t.Errorf("expected configProfile.%s in node create body, got: %s", k, captured)
		}
	}

	// Verify activeInbounds is an array (elements serialize to UUID strings).
	inbounds, ok := cp["activeInbounds"].([]any)
	if !ok {
		t.Fatalf("activeInbounds is not an array, got: %s", captured)
	}
	if len(inbounds) != 1 {
		t.Errorf("expected 1 inbound, got %d", len(inbounds))
	}
	if v, ok := inbounds[0].(string); !ok || v != "inbound-uuid-1" {
		t.Errorf("expected activeInbounds[0] = \"inbound-uuid-1\", got: %v", inbounds[0])
	}
}

// TestHostCreateContract verifies the JSON field names sent to POST /api/hosts.
// Drift in: remark, address, port, securityLayer, inbound.configProfileUuid
// would break host creation.
func TestHostCreateContract(t *testing.T) {
	var captured []byte
	server := captureRequestServer(t, http.MethodPost, "/api/hosts", &captured)
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "contract-token"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CreateHost(context.Background(), &Host{
		Remark:        "test-host",
		Address:       "host.example.com",
		Port:          443,
		SecurityLayer: "tls",
		Inbound: &HostInbound{
			ConfigProfileUUID:        "profile-uuid-456",
			ConfigProfileInboundUUID: "inbound-uuid-2",
		},
	})
	if err != nil {
		t.Fatalf("CreateHost() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(captured, &got); err != nil {
		t.Fatalf("decode request JSON: %v\nbody: %s", err, captured)
	}

	// Verify top-level fields the API requires.
	for _, k := range []string{"remark", "address", "port", "securityLayer", "inbound"} {
		if _, ok := got[k]; !ok {
			t.Errorf("expected top-level JSON key %q in host create body, got: %s", k, captured)
		}
	}

	// Verify nested inbound fields.
	inb, ok := got["inbound"].(map[string]any)
	if !ok {
		t.Fatalf("inbound is not an object, got: %s", captured)
	}
	for _, k := range []string{"configProfileUuid", "configProfileInboundUuid"} {
		if _, ok := inb[k]; !ok {
			t.Errorf("expected inbound.%s in host create body, got: %s", k, captured)
		}
	}
}

// TestUserCreateContract verifies the JSON field names sent to POST /api/users.
// Drift in: username, expireAt, trafficLimitBytes would break user creation.
func TestUserCreateContract(t *testing.T) {
	var captured []byte
	server := captureRequestServer(t, http.MethodPost, "/api/users", &captured)
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "contract-token"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CreateUser(context.Background(), &User{
		Username:          "testuser",
		ExpireAt:          "2027-01-01T00:00:00Z",
		TrafficLimitBytes: 10737418240,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(captured, &got); err != nil {
		t.Fatalf("decode request JSON: %v\nbody: %s", err, captured)
	}

	// Verify the exact field names the API expects.
	for _, k := range []string{"username", "expireAt", "trafficLimitBytes"} {
		if _, ok := got[k]; !ok {
			t.Errorf("expected JSON key %q in user create body, got: %s", k, captured)
		}
	}

	// Spot-check values to ensure correct serialization.
	if v, ok := got["username"].(string); !ok || v != "testuser" {
		t.Errorf("username = %v, want \"testuser\"", got["username"])
	}
	if v, ok := got["expireAt"].(string); !ok || v != "2027-01-01T00:00:00Z" {
		t.Errorf("expireAt = %v, want \"2027-01-01T00:00:00Z\"", got["expireAt"])
	}
}

// TestSubscriptionSettingsUpdateContract verifies the JSON field names sent to
// PATCH /api/subscription-settings. Drift in: profileTitle, supportLink would
// break subscription settings updates.
func TestSubscriptionSettingsUpdateContract(t *testing.T) {
	var captured []byte
	server := captureRequestServer(t, http.MethodPatch, "/api/subscription-settings", &captured)
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "contract-token"})
	if err != nil {
		t.Fatal(err)
	}

	profileTitle := "My Subscription"
	supportLink := "https://t.me/support"
	_, err = client.UpdateSubscriptionSettings(context.Background(), &SubscriptionSettings{
		UUID:         "settings-uuid",
		ProfileTitle: &profileTitle,
		SupportLink:  &supportLink,
	})
	if err != nil {
		t.Fatalf("UpdateSubscriptionSettings() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(captured, &got); err != nil {
		t.Fatalf("decode request JSON: %v\nbody: %s", err, captured)
	}

	// Verify the exact field names the API expects.
	for _, k := range []string{"profileTitle", "supportLink"} {
		if _, ok := got[k]; !ok {
			t.Errorf("expected JSON key %q in subscription settings update body, got: %s", k, captured)
		}
	}

	// Spot-check values.
	if v, ok := got["profileTitle"].(string); !ok || v != "My Subscription" {
		t.Errorf("profileTitle = %v, want \"My Subscription\"", got["profileTitle"])
	}
	if v, ok := got["supportLink"].(string); !ok || v != "https://t.me/support" {
		t.Errorf("supportLink = %v, want \"https://t.me/support\"", got["supportLink"])
	}
}
