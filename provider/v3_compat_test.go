package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newV3ContractClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	client.serverVersion = "3.0"
	return client
}

func decodeRequestObject(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func TestClientV3BulkUserContracts(t *testing.T) {
	t.Parallel()

	t.Run("regular action uses numeric userIds", func(t *testing.T) {
		client := newV3ContractClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/users/bulk/reset-traffic" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"userIds": []any{float64(12), float64(34)}}
			if got := decodeRequestObject(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			_, _ = io.WriteString(w, `{}`)
		})
		if err := client.BulkUserAction(context.Background(), "reset_traffic", []string{"12", "34"}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("extend uses numeric userIds", func(t *testing.T) {
		client := newV3ContractClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/users/bulk/extend-expiration-date" {
				t.Errorf("path = %s", r.URL.Path)
			}
			want := map[string]any{"userIds": []any{float64(42)}, "extendDays": float64(7)}
			if got := decodeRequestObject(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.BulkUserExtendExpiration(context.Background(), []string{"42"}, 7); err != nil {
			t.Fatal(err)
		}
	})

	client := &Client{serverVersion: "3.0"}
	if _, err := client.bulkUserBody(context.Background(), []string{"not-an-id"}); err == nil {
		t.Error("bulkUserBody accepted a non-numeric v3 identifier")
	}
}

func TestClientV3FetchUserIPsNumericUserID(t *testing.T) {
	t.Parallel()

	client := newV3ContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/connections/by-user/42":
			_, _ = io.WriteString(w, `{"response":{"jobId":"job-1"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/connections/by-user/job-1":
			_, _ = io.WriteString(w, `{"response":{"isCompleted":true,"isFailed":false,"progress":{"total":1,"completed":1,"percent":100},"result":{"success":true,"userId":42,"nodes":[{"nodeUuid":"11111111-1111-4111-8111-111111111111","nodeName":"node","countryCode":"US","ips":[{"ip":"203.0.113.7","lastSeen":"2026-08-01T00:00:00Z"}]}]}}}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	ips, err := client.FetchUserIPs(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ips, []string{"203.0.113.7"}) {
		t.Errorf("ips = %#v", ips)
	}
}

func TestClientV3DropConnectionsContracts(t *testing.T) {
	t.Parallel()

	t.Run("full schema adapts and treats 202 as success", func(t *testing.T) {
		client := newV3ContractClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/connections/drop" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{
				"dropBy":      map[string]any{"by": "userIds", "userIds": []any{float64(42)}},
				"targetNodes": map[string]any{"target": "allNodes"},
			}
			if got := decodeRequestObject(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			w.WriteHeader(http.StatusAccepted)
		})
		body := map[string]any{
			"dropBy":      map[string]any{"by": "userUuids", "userUuids": []string{"42"}},
			"targetNodes": map[string]any{"target": "allNodes"},
		}
		sent, err := client.DropConnectionsV2(context.Background(), body)
		if err != nil || !sent {
			t.Fatalf("DropConnectionsV2() = %v, %v", sent, err)
		}
		if got := body["dropBy"].(map[string]any)["by"]; got != "userUuids" {
			t.Errorf("input body was mutated: by = %v", got)
		}
	})

	t.Run("legacy helper sends v3 schema", func(t *testing.T) {
		client := newV3ContractClient(t, func(w http.ResponseWriter, r *http.Request) {
			body := decodeRequestObject(t, r)
			dropBy := body["dropBy"].(map[string]any)
			if dropBy["by"] != "userIds" || !reflect.DeepEqual(dropBy["userIds"], []any{float64(9)}) {
				t.Errorf("dropBy = %#v", dropBy)
			}
			w.WriteHeader(http.StatusAccepted)
		})
		if err := client.DropConnections(context.Background(), "9"); err != nil {
			t.Fatal(err)
		}
	})

	if _, err := adaptDropConnectionsBody(map[string]any{
		"dropBy": map[string]any{"by": "userUuids", "userUuids": []string{"invalid"}},
	}, true); err == nil {
		t.Error("adaptDropConnectionsBody accepted a non-numeric v3 identifier")
	}
}

func TestUserVersionedIdentifiersAndUpdateBody(t *testing.T) {
	t.Parallel()

	if got := userResponseIdentifier(&User{UUID: "uuid-value", ID: 42}); got != "uuid-value" {
		t.Errorf("v2 identifier = %q", got)
	}
	if got := userResponseIdentifier(&User{ID: 42}); got != "42" {
		t.Errorf("v3 identifier = %q", got)
	}

	state := &userResourceModel{UUID: types.StringValue("old-uuid"), ID: types.Int64Value(42)}
	if got, err := userStateIdentifier(context.Background(), &Client{serverVersion: "3.0"}, state); err != nil || got != "42" {
		t.Errorf("userStateIdentifier(v3) = %q, %v", got, err)
	}
	state.ID = types.Int64Null()
	if _, err := userStateIdentifier(context.Background(), &Client{serverVersion: "3.0"}, state); err == nil {
		t.Error("userStateIdentifier accepted a stale v2 UUID on v3")
	}

	plan := &userResourceModel{
		UUID:                 types.StringValue("11111111-1111-4111-8111-111111111111"),
		ID:                   types.Int64Value(42),
		Username:             types.StringValue("alice"),
		Status:               types.StringValue("ACTIVE"),
		ExpireAt:             types.StringValue("2027-01-01T00:00:00Z"),
		TrafficLimitBytes:    types.Int64Value(0),
		TrafficLimitStrategy: types.StringValue("NO_RESET"),
		ActiveInternalSquads: types.SetValueMust(types.StringType, nil),
		ExternalSquadUUID:    types.StringValue("22222222-2222-4222-8222-222222222222"),
		Description:          types.StringNull(),
		Tag:                  types.StringNull(),
		TelegramID:           types.Int64Null(),
		Email:                types.StringNull(),
		HwidDeviceLimit:      types.Int64Null(),
		TrojanPassword:       types.StringValue("secret-value"),
		VlessUUID:            types.StringValue("33333333-3333-4333-8333-333333333333"),
		SsPassword:           types.StringValue("secret-value"),
		LastTrafficResetAt:   types.StringValue("2026-01-01T00:00:00Z"),
		CreatedAt:            types.StringValue("2026-01-01T00:00:00Z"),
	}
	v3Body, err := userUpdateBody(plan, true)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(v3Body)
	if err != nil {
		t.Fatal(err)
	}
	var v3Object map[string]any
	if err := json.Unmarshal(encoded, &v3Object); err != nil {
		t.Fatal(err)
	}
	if v3Object["id"] != float64(42) {
		t.Errorf("v3 update id = %#v", v3Object["id"])
	}
	if _, ok := v3Object["uuid"]; ok {
		t.Errorf("v3 update contains removed uuid: %#v", v3Object)
	}
	if _, ok := v3Object["username"]; ok {
		t.Errorf("v3 update contains the alternative username identity: %#v", v3Object)
	}
	if v3Object["trafficLimitBytes"] != float64(0) {
		t.Errorf("v3 update lost the meaningful zero traffic limit: %#v", v3Object)
	}
	if got, ok := v3Object["activeInternalSquads"].([]any); !ok || len(got) != 0 {
		t.Errorf("v3 update did not preserve an explicit empty squad set: %#v", v3Object)
	}
	for _, key := range []string{"description", "tag", "telegramId", "email", "hwidDeviceLimit"} {
		if value, ok := v3Object[key]; !ok || value != nil {
			t.Errorf("v3 update did not preserve explicit null for %q: %#v", key, v3Object)
		}
	}
	for _, key := range []string{"trojanPassword", "vlessUuid", "ssPassword", "createdAt", "lastTrafficResetAt"} {
		if _, ok := v3Object[key]; ok {
			t.Errorf("v3 update contains create/response-only field %q", key)
		}
	}

	v2Body, err := userUpdateBody(plan, false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(v2Body)
	var v2Object map[string]any
	_ = json.Unmarshal(encoded, &v2Object)
	if v2Object["uuid"] != plan.UUID.ValueString() {
		t.Errorf("v2 update uuid = %#v", v2Object["uuid"])
	}
	if _, ok := v2Object["id"]; ok {
		t.Errorf("v2 update contains id: %#v", v2Object)
	}
	if _, ok := v2Object["username"]; ok {
		t.Errorf("v2 update contains the alternative username identity: %#v", v2Object)
	}
}

func TestSubscriptionSettingsV3CompatibilityState(t *testing.T) {
	t.Parallel()

	unknownPlan := subscriptionSettingsModel{
		ProfileTitle:                types.StringUnknown(),
		SupportLink:                 types.StringUnknown(),
		ProfileUpdateInterval:       types.Int64Unknown(),
		IsProfileWebpageURLEnabled:  types.BoolUnknown(),
		ServeJsonAtBaseSubscription: types.BoolUnknown(),
		IsShowCustomRemarks:         types.BoolUnknown(),
		HappAnnounce:                types.StringUnknown(),
		HappRouting:                 types.StringUnknown(),
		RandomizeHosts:              types.BoolUnknown(),
	}
	unknownRequest := planToSubscriptionSettings(&unknownPlan)
	if unknownRequest.ProfileTitle != nil || unknownRequest.SupportLink != nil ||
		unknownRequest.ProfileUpdateInterval != nil || unknownRequest.IsProfileWebpageURLEnabled != nil ||
		unknownRequest.ServeJsonAtBaseSubscription != nil || unknownRequest.IsShowCustomRemarks != nil ||
		unknownRequest.HappAnnounce != nil || unknownRequest.HappRouting != nil || unknownRequest.RandomizeHosts != nil {
		t.Errorf("unknown planned values leaked into subscription settings request: %#v", unknownRequest)
	}

	configured := subscriptionSettingsModel{
		ProfileTitle:               types.StringValue("Legacy title"),
		SupportLink:                types.StringValue("https://example.test/support"),
		ProfileUpdateInterval:      types.Int64Unknown(),
		IsProfileWebpageURLEnabled: types.BoolUnknown(),
		HappAnnounce:               types.StringUnknown(),
		HappRouting:                types.StringUnknown(),
	}
	current := subscriptionSettingsModel{
		ProfileTitle:               types.StringNull(),
		SupportLink:                types.StringNull(),
		ProfileUpdateInterval:      types.Int64Null(),
		IsProfileWebpageURLEnabled: types.BoolNull(),
		HappAnnounce:               types.StringNull(),
		HappRouting:                types.StringNull(),
	}
	preserveRemovedSubscriptionSettings(&configured, &current)
	if current.ProfileTitle.ValueString() != "Legacy title" || current.SupportLink.ValueString() != "https://example.test/support" {
		t.Errorf("configured values were not preserved: %#v", current)
	}
	if !current.ProfileUpdateInterval.IsNull() || !current.HappAnnounce.IsNull() {
		t.Errorf("unknown planned values leaked into state: %#v", current)
	}

	settings := planToSubscriptionSettings(&configured)
	stripRemovedSubscriptionSettings(settings)
	if settings.ProfileTitle != nil || settings.SupportLink != nil || settings.ProfileUpdateInterval != nil {
		t.Errorf("removed v3 fields remain in payload: %#v", settings)
	}
}

func TestExternalSquadVersionAdaptation(t *testing.T) {
	t.Parallel()

	squad := &ExternalSquad{
		ResponseHeaders:       json.RawMessage(`{"x-old":"value"}`),
		ResponseHeadersAdd:    json.RawMessage(`{"x-new":"value"}`),
		ResponseHeadersRemove: json.RawMessage(`["server"]`),
		SubscriptionSettings:  json.RawMessage(`{"profileTitle":"legacy","serveJsonAtBaseSubscription":true}`),
	}
	if err := adaptExternalSquadRequest(squad, true); err != nil {
		t.Fatal(err)
	}
	if len(squad.ResponseHeaders) != 0 || len(squad.ResponseHeadersAdd) == 0 || len(squad.ResponseHeadersRemove) == 0 {
		t.Errorf("v3 headers were adapted incorrectly: %#v", squad)
	}
	var filtered map[string]any
	if err := json.Unmarshal(squad.SubscriptionSettings, &filtered); err != nil {
		t.Fatal(err)
	}
	if _, ok := filtered["profileTitle"]; ok || filtered["serveJsonAtBaseSubscription"] != true {
		t.Errorf("filtered subscription settings = %#v", filtered)
	}

	previous := externalSquadModel{
		ResponseHeaders:      types.StringValue(`{"x-old":"value"}`),
		SubscriptionSettings: types.StringValue(`{"profileTitle":"legacy","serveJsonAtBaseSubscription":true}`),
	}
	current := externalSquadModel{
		ResponseHeaders:      types.StringNull(),
		SubscriptionSettings: types.StringValue(`{"serveJsonAtBaseSubscription":false}`),
	}
	if err := preserveUnsupportedExternalSquadFields(&previous, &current, true); err != nil {
		t.Fatal(err)
	}
	if current.ResponseHeaders.ValueString() != previous.ResponseHeaders.ValueString() {
		t.Errorf("legacy response_headers not preserved: %s", current.ResponseHeaders.ValueString())
	}
	var merged map[string]any
	if err := json.Unmarshal([]byte(current.SubscriptionSettings.ValueString()), &merged); err != nil {
		t.Fatal(err)
	}
	if merged["profileTitle"] != "legacy" || merged["serveJsonAtBaseSubscription"] != false {
		t.Errorf("merged subscription settings = %#v", merged)
	}

	v2Squad := &ExternalSquad{
		ResponseHeaders:       json.RawMessage(`{"x-old":"value"}`),
		ResponseHeadersAdd:    json.RawMessage(`{"x-new":"value"}`),
		ResponseHeadersRemove: json.RawMessage(`["server"]`),
	}
	if err := adaptExternalSquadRequest(v2Squad, false); err != nil {
		t.Fatal(err)
	}
	if len(v2Squad.ResponseHeaders) == 0 || len(v2Squad.ResponseHeadersAdd) != 0 || len(v2Squad.ResponseHeadersRemove) != 0 {
		t.Errorf("v2 headers were adapted incorrectly: %#v", v2Squad)
	}
	if err := adaptExternalSquadRequest(&ExternalSquad{SubscriptionSettings: json.RawMessage(`[]`)}, true); err == nil {
		t.Error("adaptExternalSquadRequest accepted non-object subscription_settings")
	}
}
