package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSnippetResourceSchemaSyncNodesOnChange(t *testing.T) {
	t.Parallel()

	resourceSchema := snippetTestSchema(t)
	attr, ok := resourceSchema.Attributes["sync_nodes_on_change"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sync_nodes_on_change attribute type = %T, want schema.BoolAttribute", resourceSchema.Attributes["sync_nodes_on_change"])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("sync_nodes_on_change must be optional-only: %#v", attr)
	}

	pendingAttr, ok := resourceSchema.Attributes["sync_pending"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("sync_pending attribute type = %T, want schema.StringAttribute", resourceSchema.Attributes["sync_pending"])
	}
	if !pendingAttr.Computed || pendingAttr.Optional || pendingAttr.Required {
		t.Fatalf("sync_pending must be computed-only: %#v", pendingAttr)
	}
}

func TestSnippetSyncVersionGateRunsBeforeMutation(t *testing.T) {
	t.Parallel()

	for _, backendVersion := range []string{"3.2.2", "3.2.3-rc.1"} {
		for _, operation := range []string{"create", "update", "delete"} {
			t.Run(backendVersion+"/"+operation, func(t *testing.T) {
				t.Parallel()

				var mutationCalls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/api/system/metadata" {
						_, _ = io.WriteString(w, `{"response":{"version":"`+backendVersion+`"}}`)
						return
					}
					mutationCalls.Add(1)
					_, _ = io.WriteString(w, `{}`)
				}))
				defer server.Close()

				client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
				if err != nil {
					t.Fatal(err)
				}
				r := &snippetResource{client: client}
				resourceSchema := snippetTestSchema(t)
				stateModel := snippetModel{
					Name:              types.StringValue("test-snippet-gate"),
					Snippet:           types.StringValue(`[{"type":"field","domain":["geosite:category-ads"]}]`),
					SyncNodesOnChange: types.BoolValue(true),
					SyncPending:       types.StringValue(snippetSyncPhaseNone),
				}
				state := snippetTestState(t, resourceSchema, stateModel)
				var hasError bool
				switch operation {
				case "create":
					plan := snippetTestPlan(t, resourceSchema, stateModel)
					var resp resource.CreateResponse
					r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)
					hasError = resp.Diagnostics.HasError()
				case "update":
					planModel := stateModel
					planModel.Snippet = types.StringValue(`[{"type":"field","domain":["geosite:google"]}]`)
					plan := snippetTestPlan(t, resourceSchema, planModel)
					resp := resource.UpdateResponse{State: snippetTestState(t, resourceSchema, planModel)}
					r.Update(t.Context(), resource.UpdateRequest{State: state, Plan: plan}, &resp)
					hasError = resp.Diagnostics.HasError()
				case "delete":
					resp := resource.DeleteResponse{State: state}
					r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
					hasError = resp.Diagnostics.HasError()
				default:
					t.Fatalf("unknown operation %q", operation)
				}

				if !hasError {
					t.Fatalf("%s diagnostics have no error, want unsupported-version error", operation)
				}
				if got := mutationCalls.Load(); got != 0 {
					t.Fatalf("%s mutation calls = %d, want 0", operation, got)
				}
			})
		}
	}
}

func TestSnippetUpdateSyncFailurePlansSyncOnlyRetry(t *testing.T) {
	t.Parallel()

	server, counts := newSnippetRecoveryServer(t, false)
	defer server.Close()
	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	r := &snippetResource{client: client}

	resourceSchema := snippetTestSchema(t)
	stateModel := snippetModel{
		Name:              types.StringValue("test-snippet"),
		Snippet:           types.StringValue(`[{"type":"field","domain":["geosite:category-ads"]}]`),
		SyncNodesOnChange: types.BoolValue(true),
		SyncPending:       types.StringValue(snippetSyncPhaseNone),
	}
	planModel := stateModel
	planModel.Snippet = types.StringValue(`[{"type":"field","domain":["geosite:category-ads","geosite:google"]}]`)
	state := snippetTestState(t, resourceSchema, stateModel)
	plan := snippetTestPlan(t, resourceSchema, planModel)

	updateResp := resource.UpdateResponse{State: snippetTestState(t, resourceSchema, planModel)}
	r.Update(t.Context(), resource.UpdateRequest{State: state, Plan: plan}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("Update diagnostics have no error, want first sync failure")
	}
	assertSnippetSyncPhase(t, updateResp.State, snippetSyncPhaseUpdate)
	if sync, update, _ := counts(); sync != 1 || update != 1 {
		t.Fatalf("calls after failed update = sync:%d update:%d, want sync:1 update:1", sync, update)
	}

	readResp := resource.ReadResponse{State: updateResp.State}
	r.Read(t.Context(), resource.ReadRequest{State: updateResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", readResp.Diagnostics)
	}
	assertSnippetSyncPhase(t, readResp.State, snippetSyncPhaseUpdate)
	if sync, update, _ := counts(); sync != 1 || update != 1 {
		t.Fatalf("Read performed a mutation: sync:%d update:%d, want sync:1 update:1", sync, update)
	}

	retryPlanModel := planModel
	retryPlanModel.SyncPending = types.StringValue(snippetSyncPhaseUpdate)
	retryPlan := snippetTestPlan(t, resourceSchema, retryPlanModel)
	modifyResp := resource.ModifyPlanResponse{Plan: retryPlan}
	r.ModifyPlan(t.Context(), resource.ModifyPlanRequest{State: readResp.State, Plan: retryPlan}, &modifyResp)
	if modifyResp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diagnostics: %v", modifyResp.Diagnostics)
	}
	var modifiedPlan snippetModel
	if diags := modifyResp.Plan.Get(t.Context(), &modifiedPlan); diags.HasError() {
		t.Fatalf("get modified plan diagnostics: %v", diags)
	}
	if got := modifiedPlan.SyncPending.ValueString(); got != snippetSyncPhaseNone {
		t.Fatalf("planned sync_pending = %q, want %q", got, snippetSyncPhaseNone)
	}

	retryResp := resource.UpdateResponse{State: snippetTestState(t, resourceSchema, modifiedPlan)}
	r.Update(t.Context(), resource.UpdateRequest{State: readResp.State, Plan: modifyResp.Plan}, &retryResp)
	if retryResp.Diagnostics.HasError() {
		t.Fatalf("retry Update diagnostics: %v", retryResp.Diagnostics)
	}
	assertSnippetSyncPhase(t, retryResp.State, snippetSyncPhaseNone)
	if sync, update, _ := counts(); sync != 2 || update != 1 {
		t.Fatalf("calls after retry = sync:%d update:%d, want sync:2 update:1", sync, update)
	}
}

func TestSnippetDeleteSyncFailurePlansSyncOnlyRetry(t *testing.T) {
	t.Parallel()

	server, counts := newSnippetRecoveryServer(t, true)
	defer server.Close()
	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	r := &snippetResource{client: client}

	resourceSchema := snippetTestSchema(t)
	stateModel := snippetModel{
		Name:              types.StringValue("test-snippet"),
		Snippet:           types.StringValue(`[{"type":"field","domain":["geosite:category-ads"]}]`),
		SyncNodesOnChange: types.BoolValue(true),
		SyncPending:       types.StringValue(snippetSyncPhaseNone),
	}
	state := snippetTestState(t, resourceSchema, stateModel)

	deleteResp := resource.DeleteResponse{State: state}
	r.Delete(t.Context(), resource.DeleteRequest{State: state}, &deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("Delete diagnostics have no error, want first sync failure")
	}
	assertSnippetSyncPhase(t, deleteResp.State, snippetSyncPhaseDelete)
	if sync, _, deleteCalls := counts(); sync != 1 || deleteCalls != 1 {
		t.Fatalf("calls after failed delete = sync:%d delete:%d, want sync:1 delete:1", sync, deleteCalls)
	}

	readResp := resource.ReadResponse{State: deleteResp.State}
	r.Read(t.Context(), resource.ReadRequest{State: deleteResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", readResp.Diagnostics)
	}
	if readResp.State.Raw.IsNull() {
		t.Fatal("Read removed resource while delete sync was still pending")
	}
	assertSnippetSyncPhase(t, readResp.State, snippetSyncPhaseDelete)
	if sync, _, deleteCalls := counts(); sync != 1 || deleteCalls != 1 {
		t.Fatalf("Read performed a mutation: sync:%d delete:%d, want sync:1 delete:1", sync, deleteCalls)
	}

	retryResp := resource.DeleteResponse{State: readResp.State}
	r.Delete(t.Context(), resource.DeleteRequest{State: readResp.State}, &retryResp)
	if retryResp.Diagnostics.HasError() {
		t.Fatalf("retry Delete diagnostics: %v", retryResp.Diagnostics)
	}
	if sync, _, deleteCalls := counts(); sync != 2 || deleteCalls != 1 {
		t.Fatalf("calls after retry = sync:%d delete:%d, want sync:2 delete:1", sync, deleteCalls)
	}
}

func TestDeleteSnippetForSyncAcceptsAlreadyDeletedSnippet(t *testing.T) {
	t.Parallel()

	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/snippets" {
			deleteCalls++
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	r := &snippetResource{client: client}
	if err := r.deleteSnippetForSync(context.Background(), "already-deleted"); err != nil {
		t.Fatalf("deleteSnippetForSync() error = %v, want nil", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
}

func snippetTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewSnippetResource().Schema(t.Context(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func snippetTestState(t *testing.T, resourceSchema schema.Schema, model snippetModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: resourceSchema}
	if diags := state.Set(t.Context(), &model); diags.HasError() {
		t.Fatalf("set state diagnostics: %v", diags)
	}
	return state
}

func snippetTestPlan(t *testing.T, resourceSchema schema.Schema, model snippetModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(t.Context(), &model); diags.HasError() {
		t.Fatalf("set plan diagnostics: %v", diags)
	}
	return plan
}

func assertSnippetSyncPhase(t *testing.T, state tfsdk.State, want string) {
	t.Helper()
	var model snippetModel
	if diags := state.Get(t.Context(), &model); diags.HasError() {
		t.Fatalf("get state diagnostics: %v", diags)
	}
	if model.SyncPending.IsNull() || model.SyncPending.IsUnknown() || model.SyncPending.ValueString() != want {
		t.Fatalf("sync_pending = %v, want %q", model.SyncPending, want)
	}
}

func newSnippetRecoveryServer(t *testing.T, snippetDeleted bool) (*httptest.Server, func() (syncCalls, updateCalls, deleteCalls int)) {
	t.Helper()
	var mu sync.Mutex
	syncCallCount := 0
	updateCallCount := 0
	deleteCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/system/metadata":
			_, _ = io.WriteString(w, `{"response":{"version":"3.2.3"}}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/snippets":
			mu.Lock()
			updateCallCount++
			mu.Unlock()
			_, _ = io.WriteString(w, `{"response":{"total":1,"snippets":[{"name":"test-snippet","snippet":[{"type":"field","domain":["geosite:category-ads","geosite:google"]}]}]}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/snippets":
			mu.Lock()
			deleteCallCount++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/snippets/actions/sync":
			mu.Lock()
			syncCallCount++
			call := syncCallCount
			mu.Unlock()
			if call == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && r.URL.Path == "/api/snippets":
			if snippetDeleted {
				_, _ = io.WriteString(w, `{"response":{"total":0,"snippets":[]}}`)
				return
			}
			_, _ = io.WriteString(w, `{"response":{"total":1,"snippets":[{"name":"test-snippet","snippet":[{"type":"field","domain":["geosite:category-ads","geosite:google"]}]}]}}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	return server, func() (int, int, int) {
		mu.Lock()
		defer mu.Unlock()
		return syncCallCount, updateCallCount, deleteCallCount
	}
}
