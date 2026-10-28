package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestSubscriptionTemplateUpdateSendsNameOnlyOnRename covers the reserved
// Default name: the backend rejects an update that carries name "Default"
// (RESERVED_TEMPLATE_NAME), so an unchanged name must not be sent.
func TestSubscriptionTemplateUpdateSendsNameOnlyOnRename(t *testing.T) {
	tests := []struct {
		name      string
		stateName string
		planName  string
		wantName  bool
	}{
		{name: "unchanged Default name is omitted", stateName: "Default", planName: "Default", wantName: false},
		{name: "unchanged custom name is omitted", stateName: "custom", planName: "custom", wantName: false},
		{name: "rename is sent", stateName: "old-name", planName: "new-name", wantName: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/system/metadata" {
					_, _ = io.WriteString(w, `{"response":{"version":"3.2.3"}}`)
					return
				}
				if r.Method != http.MethodPatch || r.URL.Path != "/api/subscription-templates" {
					t.Errorf("request = %s %s, want PATCH /api/subscription-templates", r.Method, r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				captured = body
				_, _ = io.WriteString(w, `{"response":{"uuid":"tpl-uuid","name":"`+tt.planName+`","templateType":"XRAY_JSON","templateJson":{"dns":{"servers":["1.1.1.1"]}},"encodedTemplateYaml":null}}`)
			}))
			defer server.Close()

			client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			r := &subscriptionTemplateResource{client: client}
			var schemaResp resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResp)

			model := subscriptionTemplateModel{
				UUID:         types.StringValue("tpl-uuid"),
				Name:         types.StringValue(tt.stateName),
				TemplateType: types.StringValue("XRAY_JSON"),
				TemplateJSON: types.StringValue(`{"dns":{"servers":["8.8.8.8"]}}`),
				EncodedYaml:  types.StringNull(),
			}
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(t.Context(), &model); diags.HasError() {
				t.Fatalf("set state: %v", diags)
			}
			model.Name = types.StringValue(tt.planName)
			model.TemplateJSON = types.StringValue(`{"dns":{"servers":["1.1.1.1"]}}`)
			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			if diags := plan.Set(t.Context(), &model); diags.HasError() {
				t.Fatalf("set plan: %v", diags)
			}

			resp := resource.UpdateResponse{State: state}
			r.Update(t.Context(), resource.UpdateRequest{State: state, Plan: plan}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Update diagnostics: %v", resp.Diagnostics)
			}

			var got map[string]any
			if err := json.Unmarshal(captured, &got); err != nil {
				t.Fatalf("decode request body: %v\nbody: %s", err, captured)
			}
			if _, ok := got["name"]; ok != tt.wantName {
				t.Errorf("name in request body = %v, want %v (body: %s)", ok, tt.wantName, captured)
			}
			if got["uuid"] != "tpl-uuid" {
				t.Errorf("uuid = %v, want tpl-uuid", got["uuid"])
			}
			if _, ok := got["templateJson"]; !ok {
				t.Errorf("templateJson missing from request body: %s", captured)
			}
		})
	}
}
