package provider

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBandwidthParsingHelpers(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"categories":    []any{"2026-01-01", 42, "2026-01-02"},
		"sparklineData": []any{float64(1.5), "invalid", float64(2.5)},
		"topNodes": []any{
			map[string]any{"uuid": "node-1", "color": "#fff", "name": "Node", "countryCode": "RU", "total": float64(10)},
			"invalid",
		},
		"series": []any{
			map[string]any{
				"uuid": "node-1", "name": "Node", "color": "#fff", "countryCode": "RU", "total": float64(10),
				"data": []any{float64(4), "invalid", float64(6)},
			},
			42,
		},
	}

	categories := parseCategories(data)
	if len(categories) != 2 || categories[0].ValueString() != "2026-01-01" || categories[1].ValueString() != "2026-01-02" {
		t.Errorf("categories = %#v", categories)
	}
	sparkline := parseSparklineData(data)
	if len(sparkline) != 2 || sparkline[0].ValueFloat64() != 1.5 || sparkline[1].ValueFloat64() != 2.5 {
		t.Errorf("sparkline = %#v", sparkline)
	}
	topNodes := parseTopNodes(data)
	if len(topNodes) != 1 || topNodes[0].UUID.ValueString() != "node-1" || topNodes[0].Total.ValueFloat64() != 10 {
		t.Errorf("top nodes = %#v", topNodes)
	}
	series := parseSeries(data)
	if len(series) != 1 || series[0].CountryCode.ValueString() != "RU" || len(series[0].Data) != 2 || series[0].Data[1].ValueFloat64() != 6 {
		t.Errorf("series = %#v", series)
	}

	if got := parseCategories(nil); got != nil {
		t.Errorf("empty categories = %#v, want nil", got)
	}
	if got := parseSparklineData(map[string]any{"sparklineData": "invalid"}); got != nil {
		t.Errorf("invalid sparkline = %#v, want nil", got)
	}
}

func TestStatsParsingHelpers(t *testing.T) {
	t.Parallel()

	stats := parseStatsList([]any{
		map[string]any{"tag": "inbound", "upload": "1 GB", "download": "2 GB"},
		map[string]any{"tag": nil, "upload": 42, "download": "3 GB"},
		"invalid",
	})
	if len(stats) != 2 {
		t.Fatalf("stats = %#v", stats)
	}
	if stats[0].Tag.ValueString() != "inbound" || stats[0].Upload.ValueString() != "1 GB" || stats[0].Download.ValueString() != "2 GB" {
		t.Errorf("first stat = %#v", stats[0])
	}
	if stats[1].Tag.ValueString() != "" || stats[1].Upload.ValueString() != "" || stats[1].Download.ValueString() != "3 GB" {
		t.Errorf("second stat = %#v", stats[1])
	}
	if got := parseStatsList("invalid"); got == nil || len(got) != 0 {
		t.Errorf("invalid stats = %#v, want non-nil empty slice", got)
	}

	if getString(nil) != "" || getString(42) != "" || getString("value") != "value" {
		t.Errorf("getString conversions are incorrect")
	}

	tests := []struct {
		value any
		want  float64
	}{
		{value: float64(1.5), want: 1.5},
		{value: int(2), want: 2},
		{value: int64(3), want: 3},
		{value: stringerValue("4.5"), want: 4.5},
		{value: stringerValue("invalid"), want: 0},
		{value: "6", want: 0},
		{value: nil, want: 0},
	}
	for _, tt := range tests {
		if got := getNumber(tt.value); got != tt.want {
			t.Errorf("getNumber(%#v) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

type stringerValue string

func (v stringerValue) String() string { return string(v) }

func TestCanonicalJSONPlanValues(t *testing.T) {
	t.Parallel()

	canonical, err := canonicalJSONString(" { \"z\": 1, \"a\": [ true, null ] } ")
	if err != nil || canonical != `{"a":[true,null],"z":1}` {
		t.Fatalf("canonicalJSONString() = %q, %v", canonical, err)
	}
	if _, err := canonicalJSONString("not-json"); err == nil {
		t.Fatal("canonicalJSONString accepted invalid JSON")
	}

	canonical, config, err := canonicalNodePluginJSON(`{"connectionDrop":{"enabled":false,"whitelistIps":[]}}`)
	if err != nil || canonical != `{"connectionDrop":{"enabled":false,"whitelistIps":[]},"sharedLists":[]}` {
		t.Fatalf("canonicalNodePluginJSON() = %q, %#v, %v", canonical, config, err)
	}
	if _, ok := config["sharedLists"]; !ok {
		t.Fatal("canonicalNodePluginJSON did not apply sharedLists default")
	}
	for _, invalid := range []string{"null", "[]", "not-json", `{"unsupported":true}`} {
		if _, _, err := canonicalNodePluginJSON(invalid); err == nil {
			t.Errorf("canonicalNodePluginJSON(%q) accepted a non-object", invalid)
		}
	}
}

func TestMetadataToJSON(t *testing.T) {
	t.Parallel()

	var diagnostics diag.Diagnostics
	if got := metadataToJSON(map[string]any{"metadata": map[string]any{"z": 1, "a": "value"}}, &diagnostics); got != `{"a":"value","z":1}` {
		t.Errorf("nested metadata = %q", got)
	}
	if diagnostics.HasError() {
		t.Errorf("unexpected diagnostics: %v", diagnostics)
	}

	if got := metadataToJSON(map[string]any{"key": "value"}, &diagnostics); got != `{"key":"value"}` {
		t.Errorf("raw metadata = %q", got)
	}

	diagnostics = nil
	got := metadataToJSON(map[string]any{"metadata": make(chan int), "also_bad": make(chan int)}, &diagnostics)
	if got != "{}" || !diagnostics.HasError() {
		t.Errorf("unmarshalable metadata = %q, diagnostics = %v", got, diagnostics)
	}
}

func TestHwidCreateRequest(t *testing.T) {
	t.Parallel()

	plan := &hwidDeviceModel{
		UserUUID: types.StringValue("user-id"),
		Hwid:     types.StringValue("device-id"),
		// Metadata is panel-owned/Computed: even if a value were present it must
		// never be sent on create (there is no Update endpoint and the backend
		// overwrites it on the next client connection).
		Platform:    types.StringValue("ios"),
		OsVersion:   types.StringUnknown(),
		UserAgent:   types.StringValue("app/1.0"),
		RequestIp:   types.StringNull(),
		DeviceModel: types.StringValue("phone"),
	}
	want := map[string]any{
		"userUuid": "user-id",
		"hwid":     "device-id",
	}
	if got := hwidCreateReq(plan); !reflect.DeepEqual(got, want) {
		t.Errorf("hwidCreateReq() = %#v, want %#v", got, want)
	}
}

func TestBackend28JSONContracts(t *testing.T) {
	t.Parallel()

	var user User
	if err := json.Unmarshal([]byte(`{
  "uuid":"11111111-1111-4111-8111-111111111111",
  "username":"alice",
  "expireAt":"2028-01-01T00:00:00.000Z",
  "activeInternalSquads":[{"uuid":"22222222-2222-4222-8222-222222222222","name":"default"}]
}`), &user); err != nil {
		t.Fatalf("decode Remnawave user: %v", err)
	}
	if len(user.ActiveInternalSquads) != 1 || user.ActiveInternalSquads[0].Name != "default" {
		t.Fatalf("decoded squads = %#v", user.ActiveInternalSquads)
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("encode user request: %v", err)
	}
	var userRequest map[string]any
	if err := json.Unmarshal(userJSON, &userRequest); err != nil {
		t.Fatal(err)
	}
	if got := userRequest["activeInternalSquads"].([]any)[0]; got != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("encoded activeInternalSquads[0] = %#v", got)
	}

	var node Node
	if err := json.Unmarshal([]byte(`{
  "uuid":"33333333-3333-4333-8333-333333333333",
  "name":"node","address":"node.example.com",
  "configProfile":{"activeConfigProfileUuid":"44444444-4444-4444-8444-444444444444","activeInbounds":[{"uuid":"55555555-5555-4555-8555-555555555555","tag":"VLESS","type":"vless","network":"tcp","security":"reality","port":443}]}
}`), &node); err != nil {
		t.Fatalf("decode Remnawave node: %v", err)
	}
	if len(node.ConfigProfile.ActiveInbounds) != 1 || node.ConfigProfile.ActiveInbounds[0].Tag != "VLESS" {
		t.Fatalf("decoded node inbounds = %#v", node.ConfigProfile.ActiveInbounds)
	}
	nodeJSON, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("encode node request: %v", err)
	}
	var nodeRequest map[string]any
	if err := json.Unmarshal(nodeJSON, &nodeRequest); err != nil {
		t.Fatal(err)
	}
	profile := nodeRequest["configProfile"].(map[string]any)
	if got := profile["activeInbounds"].([]any)[0]; got != "55555555-5555-4555-8555-555555555555" {
		t.Errorf("encoded activeInbounds[0] = %#v", got)
	}

	tokenJSON, err := json.Marshal(ApiToken{Name: "terraform", ExpiresInDays: 7, Scopes: []string{"*"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(tokenJSON) != `{"name":"terraform","expiresInDays":7,"scopes":["*"]}` {
		t.Errorf("API token request = %s", tokenJSON)
	}
}

func TestConfigProfileToState(t *testing.T) {
	t.Parallel()

	network := "tcp"
	security := "reality"
	port := 443
	var diagnostics diag.Diagnostics
	state := configProfileResourceModel{}
	ok := configProfileToState(&ConfigProfile{
		UUID:   "profile-id",
		Name:   "profile",
		Config: map[string]any{"log": map[string]any{"loglevel": "warning"}},
		Inbounds: []ConfigProfileInbound{{
			UUID: "inbound-id", ProfileUUID: "profile-id", Tag: "VLESS", Type: "vless",
			Network: &network, Security: &security, Port: &port, RawInbound: map[string]any{"tag": "VLESS"},
		}},
		Nodes: []ConfigProfileNode{{UUID: "node-id", Name: "node", CountryCode: "RU"}},
	}, &state, &diagnostics)
	if !ok || diagnostics.HasError() {
		t.Fatalf("configProfileToState diagnostics = %v", diagnostics)
	}
	var inbounds []configProfileInboundResourceModel
	diagnostics = state.Inbounds.ElementsAs(t.Context(), &inbounds, false)
	if diagnostics.HasError() || len(inbounds) != 1 || inbounds[0].UUID.ValueString() != "inbound-id" || inbounds[0].RawInbound.ValueString() != `{"tag":"VLESS"}` {
		t.Errorf("inbounds = %#v, diagnostics = %v", inbounds, diagnostics)
	}
	var nodes []configProfileNodeResourceModel
	diagnostics = state.Nodes.ElementsAs(t.Context(), &nodes, false)
	if diagnostics.HasError() || len(nodes) != 1 || nodes[0].UUID.ValueString() != "node-id" {
		t.Errorf("nodes = %#v, diagnostics = %v", nodes, diagnostics)
	}
}

func TestExternalSquadConversions(t *testing.T) {
	t.Parallel()

	plan := &externalSquadModel{
		Name:                 types.StringValue("external"),
		Templates:            types.StringValue(`[{"templateUuid":"template-id","templateType":"XRAY_JSON"}]`),
		SubscriptionSettings: types.StringValue(`{"profileTitle":"External"}`),
		HostOverrides:        types.StringValue(`{"vlessRouteId":7}`),
		ResponseHeaders:      types.StringValue(`{"X-Test":"value"}`),
		HwidSettings:         types.StringValue(`{"enabled":true,"fallbackDeviceLimit":2,"maxDevicesAnnounce":null}`),
		CustomRemarks:        types.StringValue(`{"expiredUsers":["expired"]}`),
		SubpageConfigUUID:    types.StringValue("subpage-id"),
	}
	squad, err := externalSquadFromPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if squad.Name != "external" || string(squad.ResponseHeaders) != `{"X-Test":"value"}` || squad.SubpageConfigUUID == nil || *squad.SubpageConfigUUID != "subpage-id" {
		t.Errorf("externalSquadFromPlan() = %#v", squad)
	}

	state := externalSquadModel{}
	squad.UUID = "squad-id"
	externalSquadToPlan(squad, &state)
	if state.UUID.ValueString() != "squad-id" || state.Templates.IsNull() || state.ResponseHeaders.ValueString() != `{"X-Test":"value"}` {
		t.Errorf("externalSquadToPlan() = %#v", state)
	}

	_, err = externalSquadFromPlan(&externalSquadModel{Name: types.StringValue("invalid"), Templates: types.StringValue("not-json")})
	if err == nil {
		t.Error("invalid templates JSON was accepted")
	}
}

func TestUserModelConversions(t *testing.T) {
	t.Parallel()

	plan := &userResourceModel{
		Username:             types.StringValue("alice"),
		Status:               types.StringValue("ACTIVE"),
		TrafficLimitBytes:    types.Int64Value(1024),
		TrafficLimitStrategy: types.StringValue("MONTH"),
		ExpireAt:             types.StringValue("2027-01-01T00:00:00Z"),
		TrojanPassword:       types.StringValue("trojan"),
		VlessUUID:            types.StringValue("vless"),
		SsPassword:           types.StringValue("shadowsocks"),
		Description:          types.StringValue("description"),
		Tag:                  types.StringValue("TAG"),
		TelegramID:           types.Int64Value(123),
		Email:                types.StringValue("alice@example.com"),
		HwidDeviceLimit:      types.Int64Value(2),
		ActiveInternalSquads: testStringSet("squad-1", "squad-2"),
		ExternalSquadUUID:    types.StringValue("external-squad"),
		CreatedAt:            types.StringValue("2026-01-01T00:00:00.000Z"),
		LastTrafficResetAt:   types.StringValue("2026-02-01T00:00:00.000Z"),
	}
	user := planToUser(plan)
	if user.Username != "alice" || user.TrafficLimitBytes != 1024 || user.Description == nil || *user.Description != "description" || user.TelegramID == nil || *user.TelegramID != 123 || len(user.ActiveInternalSquads) != 2 || user.ExternalSquadUUID == nil || *user.ExternalSquadUUID != "external-squad" || user.CreatedAt != "2026-01-01T00:00:00.000Z" || user.LastTrafficResetAt == nil {
		t.Errorf("planToUser() = %#v", user)
	}

	minimal := planToUser(&userResourceModel{
		Username:             types.StringValue("minimal"),
		ExpireAt:             types.StringValue("2027-01-01T00:00:00Z"),
		Description:          types.StringNull(),
		Tag:                  types.StringNull(),
		TelegramID:           types.Int64Null(),
		Email:                types.StringNull(),
		HwidDeviceLimit:      types.Int64Null(),
		ActiveInternalSquads: types.SetNull(types.StringType),
		ExternalSquadUUID:    types.StringNull(),
	})
	if minimal.Description != nil || minimal.Tag != nil || minimal.TelegramID != nil || minimal.Email != nil || minimal.HwidDeviceLimit != nil || minimal.ExternalSquadUUID != nil {
		t.Errorf("minimal planToUser() = %#v", minimal)
	}

	description := "server description"
	tag := "SERVER"
	telegramID := int64(456)
	email := "server@example.com"
	hwidLimit := int64(3)
	externalSquadUUID := "external-squad"
	subRevokedAt := "2026-03-01T00:00:00.000Z"
	lastTrafficResetAt := "2026-02-01T00:00:00.000Z"
	onlineAt := "2026-04-01T00:00:00.000Z"
	firstConnectedAt := "2026-01-02T00:00:00.000Z"
	lastConnectedNodeUUID := "node-id"
	state := userResourceModel{}
	userToPlan(&User{
		UUID: "user-id", ID: 7, ShortUUID: "short", Username: "alice", Status: "ACTIVE",
		TrafficLimitBytes: 2048, TrafficLimitStrategy: "MONTH", ExpireAt: "2028-01-01T00:00:00Z",
		TrojanPassword: "trojan", VlessUUID: "vless", SsPassword: "ss", SubscriptionURL: "https://sub.example.com",
		Description: &description, Tag: &tag, TelegramID: &telegramID, Email: &email, HwidDeviceLimit: &hwidLimit,
		ActiveInternalSquads: []UserSquadRef{{UUID: "squad-1", Name: "Squad"}}, ExternalSquadUUID: &externalSquadUUID,
		LastTriggeredThreshold: 80, SubRevokedAt: &subRevokedAt, LastTrafficResetAt: &lastTrafficResetAt,
		CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-04-01T00:00:00.000Z",
		UserTraffic: &UserTraffic{UsedTrafficBytes: 1024, LifetimeUsedTrafficBytes: 4096, OnlineAt: &onlineAt, FirstConnectedAt: &firstConnectedAt, LastConnectedNodeUUID: &lastConnectedNodeUUID},
	}, &state)
	if state.UUID.ValueString() != "user-id" || state.ID.ValueInt64() != 7 || state.Description.ValueString() != description || state.HwidDeviceLimit.ValueInt64() != 3 || len(state.ActiveInternalSquads.Elements()) != 1 || state.ExternalSquadUUID.ValueString() != externalSquadUUID || state.LastTriggeredThreshold.ValueInt64() != 80 || state.UsedTrafficBytes.ValueInt64() != 1024 || state.LastConnectedNodeUUID.ValueString() != "node-id" {
		t.Errorf("userToPlan() = %#v", state)
	}

	userToPlan(&User{Username: "alice"}, &state)
	if !state.Description.IsNull() || !state.Tag.IsNull() || !state.TelegramID.IsNull() || !state.Email.IsNull() || !state.HwidDeviceLimit.IsNull() {
		t.Errorf("nil server fields were not cleared: %#v", state)
	}
}

func TestNodeModelConversions(t *testing.T) {
	t.Parallel()

	inbounds := testStringSet("inbound-1", "inbound-2")
	plan := &nodeResourceModel{
		UUID:                      types.StringValue("node-id"),
		Name:                      types.StringValue("node"),
		Address:                   types.StringValue("127.0.0.1"),
		Port:                      types.Int64Value(2222),
		ProxyURL:                  types.StringValue("socks5://proxy.example.com:1080"),
		IsTrafficTrackingActive:   types.BoolValue(true),
		TrafficLimitBytes:         types.Int64Value(1024),
		TrafficResetDay:           types.Int64Value(1),
		NotifyPercent:             types.Int64Value(80),
		CountryCode:               types.StringValue("RU"),
		ConsumptionMultiplier:     types.Float64Value(1.2),
		NodeConsumptionMultiplier: types.Float64Value(1.3),
		Tags:                      testStringSet("NODE", "TEST"),
		ProviderUUID:              types.StringValue("provider-id"),
		ActivePluginUUID:          types.StringValue("plugin-id"),
		Note:                      types.StringValue("note"),
		ConfigProfileUUID:         types.StringValue("profile-id"),
		ConfigProfileInbounds:     inbounds,
	}
	node := planToNode(plan)
	if node.UUID != "node-id" || node.Port == nil || *node.Port != 2222 || node.ProxyURL == nil || *node.ProxyURL != "socks5://proxy.example.com:1080" || node.Note == nil || *node.Note != "note" || node.ConfigProfile == nil || len(node.ConfigProfile.ActiveInbounds) != 2 || node.ConsumptionMultiplier == nil || *node.ConsumptionMultiplier != 1.2 || len(node.Tags) != 2 {
		t.Errorf("planToNode() = %#v", node)
	}

	minimal := planToNode(&nodeResourceModel{
		Name:                  types.StringValue("node"),
		Address:               types.StringValue("127.0.0.1"),
		Port:                  types.Int64Null(),
		TrafficLimitBytes:     types.Int64Null(),
		TrafficResetDay:       types.Int64Null(),
		NotifyPercent:         types.Int64Null(),
		Note:                  types.StringNull(),
		ConfigProfileUUID:     types.StringNull(),
		ConfigProfileInbounds: types.SetNull(types.StringType),
	})
	if minimal.Port != nil || minimal.TrafficLimitBytes != nil || minimal.Note != nil || minimal.ConfigProfile != nil {
		t.Errorf("minimal planToNode() = %#v", minimal)
	}

	port := 2222
	traffic := int64(4096)
	reset := 10
	notify := 90
	consumption := 1.5
	nodeConsumption := 1.6
	proxyURL := "socks5://proxy.example.com:1080"
	providerUUID := "provider-id"
	pluginUUID := "plugin-id"
	note := "from server"
	lastStatusChange := "2026-04-01T00:00:00.000Z"
	lastStatusMessage := "connected"
	state := nodeResourceModel{}
	nodeToPlan(&Node{
		UUID: "node-id", Name: "node", Address: "node.example.com", Port: &port,
		IsConnected: true, IsDisabled: true, IsConnecting: false, IsTrafficTrackingActive: true,
		TrafficLimitBytes: &traffic, TrafficResetDay: &reset, NotifyPercent: &notify,
		CountryCode: "DE", Note: &note, UsersOnline: 5, ProxyURL: &proxyURL,
		ConsumptionMultiplier: &consumption, NodeConsumptionMultiplier: &nodeConsumption,
		Tags: []string{"NODE"}, ProviderUUID: &providerUUID, ActivePluginUUID: &pluginUUID,
		LastStatusChange: &lastStatusChange, LastStatusMessage: &lastStatusMessage,
		Provider: json.RawMessage(`{"uuid":"provider-id","name":"provider"}`),
		System:   json.RawMessage(`{"info":{"hostname":"node"}}`), Versions: json.RawMessage(`{"xray":"1.0","node":"2.0"}`),
		ConfigProfile: &NodeConfigProfile{ActiveConfigProfileUUID: "profile-id", ActiveInbounds: []NodeConfigProfileInbound{{UUID: "inbound-1"}}},
	}, &state)
	if state.UUID.ValueString() != "node-id" || state.Port.ValueInt64() != 2222 || state.UsersOnline.ValueInt64() != 5 || state.ConfigProfileInbounds.Elements()[0].(types.String).ValueString() != "inbound-1" || state.ProxyURL.ValueString() != proxyURL || state.ConsumptionMultiplier.ValueFloat64() != 1.5 || len(state.Tags.Elements()) != 1 || state.LastStatusMessage.ValueString() != "connected" || state.ProviderDetails.IsNull() || state.System.ValueString() != `{"info":{"hostname":"node"}}` || state.Versions.IsNull() {
		t.Errorf("nodeToPlan() = %#v", state)
	}
}

func TestHostModelConversions(t *testing.T) {
	t.Parallel()

	plan := &hostResourceModel{
		UUID:                         types.StringValue("host-id"),
		Remark:                       types.StringValue("host"),
		Address:                      types.StringValue("host.example.com"),
		Port:                         types.Int64Value(443),
		SNI:                          types.StringValue("sni.example.com"),
		HostHeader:                   types.StringValue("header.example.com"),
		ALPN:                         types.StringValue("h2"),
		Fingerprint:                  types.StringValue("chrome"),
		IsDisabled:                   types.BoolValue(true),
		SecurityLayer:                types.StringValue("TLS"),
		XHTTPExtraParams:             types.StringValue(`{"mode":"auto"}`),
		MuxParams:                    types.StringValue(`{"enabled":true}`),
		SockoptParams:                types.StringValue(`{"tcpFastOpen":true}`),
		FinalMask:                    types.StringValue(`{"enabled":false}`),
		ServerDescription:            types.StringValue("description"),
		IsHidden:                     types.BoolValue(true),
		OverrideSniFromAddress:       types.BoolValue(true),
		KeepSniBlank:                 types.BoolValue(false),
		PinnedPeerCertSha256:         types.StringValue("sha256-value"),
		VerifyPeerCertByName:         types.StringValue("peer.example.com"),
		VlessRouteID:                 types.Int64Value(42),
		ShuffleHost:                  types.BoolValue(true),
		ConfigProfileUUID:            types.StringValue("profile-id"),
		ConfigProfileInboundUUID:     types.StringValue("inbound-id"),
		Tags:                         testStringList("TAG_1", "TAG_2"),
		Nodes:                        testStringList("node-1"),
		MihomoX25519:                 types.BoolValue(true),
		MihomoIPVersion:              types.StringValue("ipv4"),
		XrayJSONTemplateUUID:         types.StringValue("template-id"),
		ExcludedInternalSquads:       testStringList("squad-1"),
		ExcludeFromSubscriptionTypes: testStringSet("MIHOMO", "SINGBOX"),
		Path:                         types.StringValue("/ws"),
	}
	host := planToHost(plan)
	if host.UUID != "host-id" || host.SNI == nil || *host.SNI != "sni.example.com" || host.Inbound == nil || host.Inbound.ConfigProfileUUID != "profile-id" || !reflect.DeepEqual(host.Tags, []string{"TAG_1", "TAG_2"}) || host.Path == nil || *host.Path != "/ws" || host.VlessRouteID == nil || *host.VlessRouteID != 42 || len(host.ExcludeFromSubscriptionTypes) != 2 {
		t.Errorf("planToHost() = %#v", host)
	}

	state := hostResourceModel{}
	hostToPlan(host, &state)
	if state.UUID.ValueString() != "host-id" || state.SNI.ValueString() != "sni.example.com" || len(state.Tags.Elements()) != 2 || state.ConfigProfileInboundUUID.ValueString() != "inbound-id" || state.Path.ValueString() != "/ws" || state.VlessRouteID.ValueInt64() != 42 || state.XHTTPExtraParams.ValueString() != `{"mode":"auto"}` || len(state.ExcludeFromSubscriptionTypes.Elements()) != 2 {
		t.Errorf("hostToPlan() = %#v", state)
	}

	hostToPlan(&Host{Remark: "minimal", Address: "host", Port: 443}, &state)
	if !state.SNI.IsNull() || !state.HostHeader.IsNull() || !state.ALPN.IsNull() || !state.Fingerprint.IsNull() || !state.ServerDescription.IsNull() {
		t.Errorf("nil host fields were not cleared: %#v", state)
	}
	if state.Tags.IsNull() || state.Tags.IsUnknown() || len(state.Tags.Elements()) != 0 {
		t.Fatalf("omitted server tags should become a known empty list, got %#v", state.Tags)
	}

	legacyTag := "LEGACY"
	legacy := hostResourceModel{Tags: types.ListUnknown(types.StringType)}
	hostToPlan(&Host{Remark: "legacy", Address: "host", Port: 443, Tag: &legacyTag}, &legacy)
	if got := legacy.Tags.Elements()[0].(types.String).ValueString(); got != legacyTag {
		t.Fatalf("legacy singular tag = %q, want %q", got, legacyTag)
	}

	minimal := hostResourceModel{
		Tags:                   types.ListUnknown(types.StringType),
		Nodes:                  types.ListUnknown(types.StringType),
		ExcludedInternalSquads: types.ListUnknown(types.StringType),
	}
	hostToPlan(&Host{Remark: "minimal", Address: "host", Port: 443}, &minimal)
	if minimal.Tags.IsNull() || minimal.Tags.IsUnknown() || len(minimal.Tags.Elements()) != 0 {
		t.Fatalf("nil tags should become a known empty list, got %#v", minimal.Tags)
	}
	if minimal.Nodes.IsNull() || minimal.Nodes.IsUnknown() || len(minimal.Nodes.Elements()) != 0 {
		t.Fatalf("nil nodes should become a known empty list, got %#v", minimal.Nodes)
	}
	if minimal.ExcludedInternalSquads.IsNull() || minimal.ExcludedInternalSquads.IsUnknown() || len(minimal.ExcludedInternalSquads.Elements()) != 0 {
		t.Fatalf("nil excluded squads should become a known empty list, got %#v", minimal.ExcludedInternalSquads)
	}
}

func TestSettingsModelConversions(t *testing.T) {
	t.Parallel()

	settingsPlan := &subscriptionSettingsModel{
		ProfileTitle:                types.StringValue("Profile"),
		SupportLink:                 types.StringValue("https://support.example.com"),
		ProfileUpdateInterval:       types.Int64Value(60),
		IsProfileWebpageURLEnabled:  types.BoolValue(true),
		ServeJsonAtBaseSubscription: types.BoolValue(true),
		IsShowCustomRemarks:         types.BoolValue(false),
		HappAnnounce:                types.StringValue("announce"),
		HappRouting:                 types.StringValue("routing"),
		RandomizeHosts:              types.BoolValue(true),
		CustomRemarks:               types.StringValue(`{"expiredUsers":["expired"]}`),
		CustomResponseHeaders:       types.StringValue(`{"X-Test":"value"}`),
		ResponseRules:               types.StringValue(`{"version":"1","settings":{},"rules":[]}`),
		HwidSettings:                types.StringValue(`{"enabled":true,"fallbackDeviceLimit":2,"maxDevicesAnnounce":null}`),
	}
	settings := planToSubscriptionSettings(settingsPlan)
	if settings.ProfileTitle == nil || *settings.ProfileTitle != "Profile" || settings.ProfileUpdateInterval == nil || *settings.ProfileUpdateInterval != 60 || settings.RandomizeHosts == nil || !*settings.RandomizeHosts || string(settings.HwidSettings) == "" {
		t.Errorf("planToSubscriptionSettings() = %#v", settings)
	}

	state := subscriptionSettingsModel{}
	subscriptionSettingsToPlan(settings, &state)
	if state.ProfileTitle.ValueString() != "Profile" || state.ProfileUpdateInterval.ValueInt64() != 60 || !state.RandomizeHosts.ValueBool() || state.CustomResponseHeaders.ValueString() != `{"X-Test":"value"}` || state.HwidSettings.IsNull() {
		t.Errorf("subscriptionSettingsToPlan() = %#v", state)
	}

	panelPlan := &panelSettingsModel{
		BrandingTitle:       types.StringValue("Panel"),
		BrandingLogoURL:     types.StringValue("https://example.com/logo.svg"),
		PasswordAuthEnabled: types.BoolValue(true),
		PasskeySettings:     types.StringValue(`{"enabled":true}`),
		OAuth2Settings:      types.StringValue(`{"provider":"example"}`),
	}
	panel := planToPanelSettings(panelPlan)
	if panel.BrandingSettings == nil || panel.BrandingSettings.Title == nil || *panel.BrandingSettings.Title != "Panel" || panel.PasswordSettings == nil || panel.PasskeySettings == nil || panel.OAuth2Settings == nil {
		t.Errorf("planToPanelSettings() = %#v", panel)
	}

	panelState := panelSettingsModel{}
	panelSettingsToPlan(panel, &panelState)
	if panelState.BrandingTitle.ValueString() != "Panel" || !panelState.PasswordAuthEnabled.ValueBool() || panelState.PasskeySettings.ValueString() != `{"enabled":true}` || panelState.OAuth2Settings.ValueString() != `{"provider":"example"}` {
		t.Errorf("panelSettingsToPlan() = %#v", panelState)
	}

	nullState := panelSettingsModel{
		BrandingTitle:       types.StringUnknown(),
		BrandingLogoURL:     types.StringUnknown(),
		PasswordAuthEnabled: types.BoolUnknown(),
		PasskeySettings:     types.StringUnknown(),
		OAuth2Settings:      types.StringUnknown(),
	}
	panelSettingsToPlan(&PanelSettings{BrandingSettings: &BrandingSettings{}}, &nullState)
	if !nullState.BrandingTitle.IsNull() || !nullState.BrandingLogoURL.IsNull() || !nullState.PasswordAuthEnabled.IsNull() || !nullState.PasskeySettings.IsNull() || !nullState.OAuth2Settings.IsNull() {
		t.Errorf("null panel settings were not resolved: %#v", nullState)
	}

	invalidJSON := planToPanelSettings(&panelSettingsModel{
		PasskeySettings: types.StringValue("not-json"),
		OAuth2Settings:  types.StringValue("not-json"),
	})
	if invalidJSON.PasskeySettings != nil || invalidJSON.OAuth2Settings != nil {
		t.Errorf("invalid JSON settings = %#v", invalidJSON)
	}

	unknown := planToPanelSettings(&panelSettingsModel{
		BrandingTitle:       types.StringUnknown(),
		BrandingLogoURL:     types.StringUnknown(),
		PasswordAuthEnabled: types.BoolUnknown(),
		PasskeySettings:     types.StringUnknown(),
		OAuth2Settings:      types.StringUnknown(),
	})
	if unknown.BrandingSettings != nil || unknown.PasswordSettings != nil || unknown.PasskeySettings != nil || unknown.OAuth2Settings != nil {
		t.Errorf("unknown settings must be omitted: %#v", unknown)
	}
}

func TestMergePanelSettings(t *testing.T) {
	t.Parallel()

	oldTitle := "Old title"
	newTitle := "New title"
	logoURL := "https://example.com/logo.svg"
	current := &PanelSettings{BrandingSettings: &BrandingSettings{Title: &oldTitle, LogoURL: &logoURL}}
	planned := &PanelSettings{BrandingSettings: &BrandingSettings{Title: &newTitle}}

	merged := mergePanelSettings(current, planned)
	if merged.BrandingSettings.Title == nil || *merged.BrandingSettings.Title != newTitle {
		t.Errorf("planned title was not preserved: %#v", merged)
	}
	if merged.BrandingSettings.LogoURL == nil || *merged.BrandingSettings.LogoURL != logoURL {
		t.Errorf("current logo URL was not merged: %#v", merged)
	}

	withoutBranding := &PanelSettings{}
	if got := mergePanelSettings(current, withoutBranding); got != withoutBranding || got.BrandingSettings != nil {
		t.Errorf("unexpected branding merge: %#v", got)
	}
	if got := mergePanelSettings(nil, planned); got != planned {
		t.Errorf("nil current settings changed planned settings")
	}
}

func TestBrandingSettingsMarshalNullFields(t *testing.T) {
	t.Parallel()

	title := "Panel"
	settings := &PanelSettings{BrandingSettings: &BrandingSettings{Title: &title}}
	got, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(got) != `{"brandingSettings":{"title":"Panel","logoUrl":null}}` {
		t.Fatalf("json.Marshal() = %s", got)
	}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want bool
	}{
		{err: nil, want: false},
		{err: errors.New("record not found"), want: true},
		{err: &HTTPStatusError{StatusCode: 404}, want: true},
		{err: &HTTPStatusError{StatusCode: 500, Body: "failure"}, want: false},
	}
	for _, tt := range tests {
		if got := isNotFound(tt.err); got != tt.want {
			t.Errorf("isNotFound(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func testStringList(values ...string) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	result, diagnostics := types.ListValue(types.StringType, elements)
	if diagnostics.HasError() {
		panic(diagnostics.Errors())
	}
	return result
}

func testStringSet(values ...string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	result, diagnostics := types.SetValue(types.StringType, elements)
	if diagnostics.HasError() {
		panic(diagnostics.Errors())
	}
	return result
}
