package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// nodeActionResource is an imperative (run-once) resource that triggers a
// per-node action on the Remnawave backend: enable, disable, restart or
// reset_traffic. Because the action is idempotent on the server, the resource
// simply executes the action on Create and whenever `triggers` changes,
// and is a no-op on Read/Delete.
//
// "reset-traffic" (hyphen) is accepted as a backward-compatible alias for
// the canonical "reset_traffic" form. Using the hyphenated form emits a
// deprecation warning via tflog.Warn.
type nodeActionResource struct {
	client *Client
}

// nodeActionAliases maps deprecated/alias action names to their canonical
// (underscore) form.
var nodeActionAliases = map[string]string{
	"reset-traffic": "reset_traffic",
}

// normalizeNodeAction returns the canonical form of action. If action is a
// known alias (e.g. "reset-traffic"), the canonical name is returned and
// warned is set to true so the caller can emit a deprecation warning.
func normalizeNodeAction(action string) (canonical string, warned bool) {
	if c, ok := nodeActionAliases[action]; ok {
		return c, true
	}
	return action, false
}

type nodeActionResourceModel struct {
	ID           types.String `tfsdk:"id"`
	NodeUUID     types.String `tfsdk:"node_uuid"`
	Action       types.String `tfsdk:"action"`
	ForceRestart types.Bool   `tfsdk:"force_restart"`
	Triggers     types.List   `tfsdk:"triggers"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func NewNodeActionResource() resource.Resource {
	return &nodeActionResource{}
}

func (r *nodeActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node_action"
}

func (r *nodeActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Executes a one-time action on a Remnawave node (enable, disable, restart, or reset_traffic). Use `triggers` to force re-execution when values change. `reset-traffic` is accepted as a backward-compatible alias for `reset_traffic` (prefer the underscore form).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Opaque identifier for this action invocation. Always set to the node UUID.",
			},
			"node_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the target node.",
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: "Action to perform. One of: `enable`, `disable`, `restart`, `reset_traffic`. `reset-traffic` is accepted as a backward-compatible alias for `reset_traffic`.",
			},
			"force_restart": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to force-restart the node (only meaningful for action = `restart`). Defaults to `false`.",
			},
			"triggers": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary list of strings. When any value changes, the action is re-executed. Use `timestamp()` to force a re-run on every apply.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when the action was last executed.",
			},
		},
	}
}

func (r *nodeActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ─── CRUD ───

func (r *nodeActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nodeActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := r.executeAction(ctx, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nodeActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the node still exists so the resource can be recreated if the
	// underlying node was deleted.
	nodeUUID := state.NodeUUID.ValueString()
	if nodeUUID == "" {
		return
	}
	if _, err := r.client.GetNodeByUUID(ctx, nodeUUID); err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "node not found, removing node_action from state", map[string]any{"node_uuid": nodeUUID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read node for node_action", err.Error())
		return
	}
}

func (r *nodeActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nodeActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-execute the action whenever any input changes (node_uuid, action,
	// force_restart, or triggers).
	if diags := r.executeAction(ctx, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: there is nothing to undo for an imperative action.
}

// ─── Helpers ───

func (r *nodeActionResource) executeAction(ctx context.Context, m *nodeActionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	nodeUUID := m.NodeUUID.ValueString()
	action := m.Action.ValueString()

	// Normalize alias spellings (e.g. "reset-traffic" → "reset_traffic") to
	// the canonical underscore form and warn about deprecated usage.
	canonicalAction, warnAlias := normalizeNodeAction(action)
	if warnAlias {
		tflog.Warn(ctx, "node_action uses deprecated hyphenated form; prefer the underscore form (will keep working but may be removed in a future release)", map[string]any{
			"action":           action,
			"canonical_action": canonicalAction,
			"node_uuid":        nodeUUID,
		})
		action = canonicalAction
		m.Action = types.StringValue(canonicalAction)
	}

	switch action {
	case "enable":
		if _, err := r.client.EnableNode(ctx, nodeUUID); err != nil {
			diags.AddError("Failed to enable node", err.Error())
			return diags
		}
	case "disable":
		if _, err := r.client.DisableNode(ctx, nodeUUID); err != nil {
			diags.AddError("Failed to disable node", err.Error())
			return diags
		}
	case "restart":
		force := false
		if !m.ForceRestart.IsNull() && !m.ForceRestart.IsUnknown() {
			force = m.ForceRestart.ValueBool()
		}
		if _, err := r.client.RestartNode(ctx, nodeUUID, force); err != nil {
			diags.AddError("Failed to restart node", err.Error())
			return diags
		}
	case "reset_traffic":
		if _, err := r.client.ResetNodeTraffic(ctx, nodeUUID); err != nil {
			diags.AddError("Failed to reset node traffic", err.Error())
			return diags
		}
	default:
		diags.AddError("Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, restart, reset_traffic; got %q", action))
		return diags
	}

	m.ID = types.StringValue(nodeUUID)
	m.CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	if m.ForceRestart.IsNull() || m.ForceRestart.IsUnknown() {
		m.ForceRestart = types.BoolValue(false)
	}

	tflog.Info(ctx, "node action executed", map[string]any{
		"node_uuid": nodeUUID,
		"action":    action,
	})
	return diags
}
