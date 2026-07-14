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
	name     string
	method   string
	path     string
	query    map[string]string
	args     []any
	wantJSON map[string]any
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
		{name: "GetAllUsers", method: http.MethodGet, path: "/api/users"},

		{name: "CreateNode", method: http.MethodPost, path: "/api/nodes", args: []any{&Node{Name: "node", Address: "127.0.0.1"}}},
		{name: "GetAllNodes", method: http.MethodGet, path: "/api/nodes"},
		{name: "GetNodeByUUID", method: http.MethodGet, path: "/api/nodes/item-id", args: []any{"item-id"}},
		{name: "UpdateNode", method: http.MethodPatch, path: "/api/nodes", args: []any{&Node{UUID: "item-id", Name: "node", Address: "127.0.0.1"}}},
		{name: "DeleteNode", method: http.MethodDelete, path: "/api/nodes/item-id", args: []any{"item-id"}},

		{name: "CreateHost", method: http.MethodPost, path: "/api/hosts", args: []any{&Host{Remark: "host", Address: "host.example.com", Port: 443}}},
		{name: "GetAllHosts", method: http.MethodGet, path: "/api/hosts"},
		{name: "GetHostByUUID", method: http.MethodGet, path: "/api/hosts/item-id", args: []any{"item-id"}},
		{name: "UpdateHost", method: http.MethodPatch, path: "/api/hosts", args: []any{&Host{UUID: "item-id", Remark: "host", Address: "host.example.com", Port: 443}}},
		{name: "DeleteHost", method: http.MethodDelete, path: "/api/hosts/item-id", args: []any{"item-id"}},
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
		{name: "DeleteBillingHistory", method: http.MethodDelete, path: "/api/infra-billing/history/item-id", args: []any{"item-id"}},

		{name: "GetSubscriptionByUUID", method: http.MethodGet, path: "/api/subscriptions/by-uuid/item-id", args: []any{"item-id"}},
		{name: "GetSubscriptionByUsername", method: http.MethodGet, path: "/api/subscriptions/by-username/alice", args: []any{"alice"}},
		{name: "GetSubscriptionByShortUUID", method: http.MethodGet, path: "/api/subscriptions/by-short-uuid/short-id", args: []any{"short-id"}},
		{name: "GetSubscriptionRequestHistory", method: http.MethodGet, path: "/api/subscription-request-history"},

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
		{name: "GetUserHwidDevices", method: http.MethodGet, path: "/api/hwid/devices/item-id", args: []any{"item-id"}},
		{name: "GetHwidStats", method: http.MethodGet, path: "/api/hwid/devices/stats"},
		{name: "GetHwidTopUsers", method: http.MethodGet, path: "/api/hwid/devices/top-users"},
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
				wantBody := tt.method == http.MethodPost || tt.method == http.MethodPatch || tt.method == http.MethodPut || tt.wantJSON != nil
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
				_, _ = io.WriteString(w, `{"response":`+response+`}`)
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
	default:
		return caseName
	}
}
