package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type snippetResource struct{ client *Client }

const (
	snippetSyncPhaseNone     = "none"
	snippetSyncPhaseUpdate   = "update"
	snippetSyncPhaseDelete   = "delete"
	snippetSyncPhaseRecreate = "recreate"
)

type snippetModel struct {
	Name              types.String `tfsdk:"name"`
	Snippet           types.String `tfsdk:"snippet"`
	SyncNodesOnChange types.Bool   `tfsdk:"sync_nodes_on_change"`
	SyncPending       types.String `tfsdk:"sync_pending"`
}

var _ resource.ResourceWithModifyPlan = (*snippetResource)(nil)

func NewSnippetResource() resource.Resource { return &snippetResource{} }

func (r *snippetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_snippet"
}

func (r *snippetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave Xray config snippet (keyed by name).",
		Attributes: map[string]schema.Attribute{
			"name":    schema.StringAttribute{Required: true, Description: "Snippet name (2-255 chars)."},
			"snippet": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Snippet content as JSON array string."},
			"sync_nodes_on_change": schema.BoolAttribute{
				Optional: true,
				Description: "Restart nodes using config profiles that reference this snippet after update or deletion. " +
					"Requires Remnawave 3.2.3+, system:metadata (or broader system:read) for version detection, and snippets:sync " +
					"in addition to the snippets:list/create/update/delete scopes used by normal resource lifecycle operations. " +
					"Defaults to false because synchronization is disruptive.",
			},
			"sync_pending": schema.StringAttribute{
				Computed: true,
				Description: "Provider-managed recovery phase: none, update, delete, or recreate. A pending phase means an opt-in mutation may have " +
					"reached the backend but node synchronization did not complete. Refresh remains read-only and preserves a reachable " +
					"sync-only plan; recovery is at-least-once, so an ambiguous transport failure can cause a duplicate restart.",
			},
		},
	}
}

func (r *snippetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	r.client = client
}

func (r *snippetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan snippetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var snippetData any
	if err := json.Unmarshal([]byte(plan.Snippet.ValueString()), &snippetData); err != nil {
		resp.Diagnostics.AddError("Invalid snippet JSON", err.Error())
		return
	}
	if plan.SyncNodesOnChange.ValueBool() {
		if err := r.requireSyncSupport(ctx); err != nil {
			resp.Diagnostics.AddError("Unsupported snippet synchronization", err.Error())
			return
		}
	}
	_, err := r.client.CreateSnippet(ctx, &Snippet{Name: plan.Name.ValueString(), Snippet: snippetData})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create snippet", err.Error())
		return
	}
	plan.SyncPending = types.StringValue(snippetSyncPhaseNone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snippetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	phase, err := snippetSyncPhase(state.SyncPending)
	if err != nil {
		resp.Diagnostics.AddError("Invalid snippet synchronization state", err.Error())
		return
	}

	list, err := r.client.GetSnippets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read snippets", err.Error())
		return
	}
	var remoteSnippet types.String
	found := false
	for _, s := range list.Snippets {
		if s.Name != state.Name.ValueString() {
			continue
		}
		found = true
		b, err := json.Marshal(s.Snippet)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal snippet", err.Error())
			return
		}
		remoteSnippet = types.StringValue(string(b))
		break
	}

	switch phase {
	case snippetSyncPhaseUpdate:
		// A failed or ambiguous PATCH is pending. If the desired payload reached
		// the backend, preserve the phase so ModifyPlan schedules sync-only. If it
		// did not, accept the remote payload and let normal drift retry PATCH+sync.
		if !found {
			state.SyncPending = types.StringValue(snippetSyncPhaseRecreate)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		if snippetJSONEqual(remoteSnippet.ValueString(), state.Snippet.ValueString()) {
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		state.Snippet = remoteSnippet
		state.SyncPending = types.StringValue(snippetSyncPhaseNone)
	case snippetSyncPhaseDelete:
		// Preserve state when DELETE committed but sync failed, keeping Delete
		// reachable. If the object still exists, retry the primary mutation.
		if !found {
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		state.Snippet = remoteSnippet
		state.SyncPending = types.StringValue(snippetSyncPhaseNone)
	case snippetSyncPhaseRecreate:
		if !found {
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		if snippetJSONEqual(remoteSnippet.ValueString(), state.Snippet.ValueString()) {
			state.SyncPending = types.StringValue(snippetSyncPhaseUpdate)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		state.Snippet = remoteSnippet
		state.SyncPending = types.StringValue(snippetSyncPhaseNone)
	case snippetSyncPhaseNone:
		if !found {
			resp.State.RemoveResource(ctx)
			return
		}
		state.Snippet = remoteSnippet
		state.SyncPending = types.StringValue(snippetSyncPhaseNone)
	default:
		resp.Diagnostics.AddError("Invalid snippet synchronization state", fmt.Sprintf("unexpected sync_pending phase %q", phase))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *snippetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	phase, err := snippetSyncPhase(state.SyncPending)
	if err != nil {
		resp.Diagnostics.AddError("Invalid snippet synchronization state", err.Error())
		return
	}
	if phase == snippetSyncPhaseNone {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("sync_pending"), types.StringValue(snippetSyncPhaseNone))...)
}

func (r *snippetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var plan snippetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	phase, err := snippetSyncPhase(state.SyncPending)
	if err != nil {
		resp.Diagnostics.AddError("Invalid snippet synchronization state", err.Error())
		return
	}
	recreate := phase == snippetSyncPhaseRecreate || phase == snippetSyncPhaseDelete
	if phase != snippetSyncPhaseNone {
		if err := r.requireSyncSupport(ctx); err != nil {
			resp.Diagnostics.AddError("Unsupported snippet synchronization", err.Error())
			return
		}
		plan.SyncPending = types.StringValue(phase)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SyncSnippet(ctx, state.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to synchronize snippet nodes", err.Error())
			return
		}
		plan.SyncPending = types.StringValue(snippetSyncPhaseNone)
		if !recreate && snippetBusinessStateEqual(state, plan) {
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			return
		}
	}

	var snippetData any
	if err := json.Unmarshal([]byte(plan.Snippet.ValueString()), &snippetData); err != nil {
		resp.Diagnostics.AddError("Invalid snippet JSON", err.Error())
		return
	}
	if plan.SyncNodesOnChange.ValueBool() {
		if err := r.requireSyncSupport(ctx); err != nil {
			resp.Diagnostics.AddError("Unsupported snippet synchronization", err.Error())
			return
		}
		plan.SyncPending = types.StringValue(snippetSyncPhaseUpdate)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if recreate {
		_, err = r.client.CreateSnippet(ctx, &Snippet{Name: plan.Name.ValueString(), Snippet: snippetData})
	} else {
		_, err = r.client.UpdateSnippet(ctx, &Snippet{Name: plan.Name.ValueString(), Snippet: snippetData})
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to update snippet", err.Error())
		return
	}
	if plan.SyncNodesOnChange.ValueBool() {
		if err := r.client.SyncSnippet(ctx, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to synchronize snippet nodes", err.Error())
			return
		}
	}
	plan.SyncPending = types.StringValue(snippetSyncPhaseNone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snippetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := state.Name.ValueString()
	phase, err := snippetSyncPhase(state.SyncPending)
	if err != nil {
		resp.Diagnostics.AddError("Invalid snippet synchronization state", err.Error())
		return
	}
	if phase != snippetSyncPhaseNone {
		if err := r.requireSyncSupport(ctx); err != nil {
			resp.Diagnostics.AddError("Unsupported snippet synchronization", err.Error())
			return
		}
		state.SyncPending = types.StringValue(phase)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SyncSnippet(ctx, name); err != nil {
			resp.Diagnostics.AddError("Failed to synchronize snippet nodes", err.Error())
			return
		}
		if phase == snippetSyncPhaseDelete || phase == snippetSyncPhaseRecreate {
			return
		}
		state.SyncPending = types.StringValue(snippetSyncPhaseNone)
	}

	if state.SyncNodesOnChange.ValueBool() {
		if err := r.requireSyncSupport(ctx); err != nil {
			resp.Diagnostics.AddError("Unsupported snippet synchronization", err.Error())
			return
		}
		state.SyncPending = types.StringValue(snippetSyncPhaseDelete)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.deleteSnippetForSync(ctx, name); err != nil {
			resp.Diagnostics.AddError("Failed to delete snippet", err.Error())
			return
		}
		if err := r.client.SyncSnippet(ctx, name); err != nil {
			resp.Diagnostics.AddError("Failed to synchronize snippet nodes", err.Error())
		}
		return
	}
	if err := r.client.DeleteSnippet(ctx, name); err != nil {
		resp.Diagnostics.AddError("Failed to delete snippet", err.Error())
	}
}

func (r *snippetResource) requireSyncSupport(ctx context.Context) error {
	supported, err := r.client.isVersionAtLeast3_2_3(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine backend version: %w", err)
	}
	if !supported {
		return fmt.Errorf("sync_nodes_on_change requires Remnawave 3.2.3 or later")
	}
	return nil
}

func snippetSyncPhase(value types.String) (string, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return snippetSyncPhaseNone, nil
	}
	phase := value.ValueString()
	switch phase {
	case snippetSyncPhaseNone, snippetSyncPhaseUpdate, snippetSyncPhaseDelete, snippetSyncPhaseRecreate:
		return phase, nil
	default:
		return "", fmt.Errorf("unexpected sync_pending phase %q", phase)
	}
}

func snippetJSONEqual(left, right string) bool {
	canonicalLeft, err := canonicalJSONString(left)
	if err != nil {
		return false
	}
	canonicalRight, err := canonicalJSONString(right)
	if err != nil {
		return false
	}
	return canonicalLeft == canonicalRight
}

func snippetBusinessStateEqual(state, plan snippetModel) bool {
	return state.Name.ValueString() == plan.Name.ValueString() &&
		snippetJSONEqual(state.Snippet.ValueString(), plan.Snippet.ValueString()) &&
		state.SyncNodesOnChange.ValueBool() == plan.SyncNodesOnChange.ValueBool()
}

// deleteSnippetForSync treats an already-deleted snippet as success so a
// destroy can safely retry after DELETE succeeded but the subsequent sync call
// failed. The sync endpoint resolves affected profiles by snippet name and does
// not require the snippet record to still exist.
func (r *snippetResource) deleteSnippetForSync(ctx context.Context, name string) error {
	err := r.client.DeleteSnippet(ctx, name)
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func (r *snippetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(req.ID))...)
}
