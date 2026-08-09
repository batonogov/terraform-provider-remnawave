package provider

import (
	"reflect"
	"strings"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNodeIPAddressValidator(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "ipv4", value: "192.0.2.10"},
		{name: "ipv6", value: "2001:db8::10"},
		{name: "hostname", value: "node.example.com", wantErr: true},
		{name: "malformed", value: "999.0.2.10", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request := frameworkvalidator.StringRequest{
				Path:        path.Root("ip"),
				ConfigValue: types.StringValue(tt.value),
			}
			var response frameworkvalidator.StringResponse
			nodeIPAddressValidator{}.ValidateString(t.Context(), request, &response)
			if response.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("ValidateString(%q) diagnostics = %v, wantErr %v", tt.value, response.Diagnostics, tt.wantErr)
			}
		})
	}
}

func TestNodeIPStatusesMatchBackendContract(t *testing.T) {
	t.Parallel()

	want := []string{"INBOUND", "OUTBOUND", "MANAGEMENT", "TRANSIT", "MONITORING", "RESERVE", "BLOCKED", "FLAGGED", "DEPRECATED", "UNKNOWN"}
	if !reflect.DeepEqual(nodeIPStatuses, want) {
		t.Fatalf("nodeIPStatuses = %v, want %v", nodeIPStatuses, want)
	}
}

func TestNodesDataSourceSchemaIncludesIPs(t *testing.T) {
	t.Parallel()

	schema := nodesDataSourceSchema()

	nodes, ok := schema.Attributes["nodes"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("nodes attribute type = %T", schema.Attributes["nodes"])
	}
	ips, ok := nodes.NestedObject.Attributes["ips"].(datasourceschema.SetNestedAttribute)
	if !ok || !ips.IsComputed() {
		t.Fatalf("nodes.ips attribute = %#v, want computed nested set", nodes.NestedObject.Attributes["ips"])
	}
}

func TestNodeToItemIncludesIPs(t *testing.T) {
	t.Parallel()

	backendIPs := []NodeIP{
		{IP: "192.0.2.10", Status: "MANAGEMENT"},
		{IP: "2001:db8::10", Status: "INBOUND"},
	}
	item, diagnostics := nodeToItem(t.Context(), Node{
		UUID: "node-id",
		Name: "node",
		IPs:  &backendIPs,
	})
	if diagnostics.HasError() {
		t.Fatalf("nodeToItem diagnostics = %v", diagnostics)
	}
	var ips []nodeIPResourceModel
	diagnostics = item.IPs.ElementsAs(t.Context(), &ips, false)
	if diagnostics.HasError() {
		t.Fatalf("decode item IPs: %v", diagnostics)
	}
	gotIPs := make(map[string]string, len(ips))
	for _, item := range ips {
		gotIPs[item.IP.ValueString()] = item.Status.ValueString()
	}
	if len(ips) != 2 || gotIPs["192.0.2.10"] != "MANAGEMENT" || gotIPs["2001:db8::10"] != "INBOUND" {
		t.Fatalf("item IPs = %#v", ips)
	}
}

func TestNodeIPsToSetRejectsInvalidBackendData(t *testing.T) {
	t.Parallel()

	tooMany := make([]NodeIP, 65)
	for i := range tooMany {
		tooMany[i] = NodeIP{IP: "192.0.2.10", Status: "INBOUND"}
	}

	for _, tt := range []struct {
		name string
		ips  []NodeIP
	}{
		{name: "too many", ips: tooMany},
		{name: "oversized ip", ips: []NodeIP{{IP: strings.Repeat("1", 46), Status: "INBOUND"}}},
		{name: "malformed ip", ips: []NodeIP{{IP: "999.0.2.10", Status: "INBOUND"}}},
		{name: "unknown status", ips: []NodeIP{{IP: "192.0.2.10", Status: "UNTRUSTED"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ips := tt.ips
			_, diagnostics := nodeIPsToSet(t.Context(), &ips)
			if !diagnostics.HasError() {
				t.Fatal("nodeIPsToSet() diagnostics have no error")
			}
		})
	}
}
