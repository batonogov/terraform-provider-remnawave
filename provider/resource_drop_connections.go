package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dropConnectionsResource is an imperative resource that triggers a
// "drop connections" action on the Remnawave IP Control module when applied.
// It supports dropping by user UUIDs or by IP addresses, and optionally
// targeting specific nodes.
type dropConnectionsResource struct {
	client *Client
}

type dropConnectionsModel struct {
	ID          types.String `tfsdk:"id"`
	DropBy      types.String `tfsdk:"drop_by"`
	UserUUIDs   types.List   `tfsdk:"user_uuids"`
	IPAddresses types.List   `tfsdk:"ip_addresses"`
	Target      types.String `tfsdk:"target"`
	NodeUUIDs   types.List   `tfsdk:"node_uuids"`
	Triggers    types.Map    `tfsdk:"triggers"`
	EventSent   types.Bool   `tfsdk:"event_sent"`
}

func NewDropConnectionsResource() resource.Resource {
	return &dropConnectionsResource{}
}

func (r *dropConnectionsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_drop_connections"
}

func (r *dropConnectionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Drops active connections for users or IP addresses via the Remnawave IP Control module. " +
			"Supports targeting all nodes or specific nodes. This is an imperative action resource: " +
			"applying it sends a drop-connections request to the panel. " +
			"Use the optional triggers map to force re-execution when its values change.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Stable identifier derived from input values and triggers.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"drop_by": schema.StringAttribute{
				Required: true,
				Description: "Drop mode: either `user_uuids` or `ip_addresses`. " +
					"When `user_uuids`, provide `user_uuids` list. When `ip_addresses`, provide `ip_addresses` list.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_uuids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of user UUIDs to drop connections for. Required when drop_by = user_uuids.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"ip_addresses": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of IP addresses to drop connections for. Required when drop_by = ip_addresses.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"target": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Node targeting: `all_nodes` (default) or `specific_nodes`. " +
					"When `specific_nodes`, provide `node_uuids` list.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"node_uuids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of node UUIDs to target. Required when target = specific_nodes.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A map of arbitrary string values. When any value changes, the resource is replaced and the drop-connections action is re-executed.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"event_sent": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the drop-connections event was sent successfully.",
			},
		},
	}
}

func (r *dropConnectionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dropConnectionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dropConnectionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := buildDropConnectionsBody(&plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	eventSent, err := r.client.DropConnectionsV2(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to drop connections", err.Error())
		return
	}

	plan.ID = types.StringValue(computeDropConnectionsID(&plan))
	plan.EventSent = types.BoolValue(eventSent)
	if plan.Target.IsNull() || plan.Target.IsUnknown() {
		plan.Target = types.StringValue("all_nodes")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dropConnectionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dropConnectionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Nothing to read from the backend — this is a fire-and-forget action.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dropConnectionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace; Update should never be called.
	resp.Diagnostics.AddError("Unexpected update", "All attributes require replacement.")
}

func (r *dropConnectionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do: the action was already executed at apply time and there
	// is no "undo" on the backend. We simply remove the resource from state.
	var state dropConnectionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dropConnectionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "remnawave_drop_connections is an imperative action resource and cannot be imported.")
}

// ─── Helpers ───

// buildDropConnectionsBody constructs the API request body from the Terraform model.
// Backend schema: { dropBy: { by: "userUuids"|"ipAddresses", ... }, targetNodes: { target: "allNodes"|"specificNodes", ... } }
func buildDropConnectionsBody(m *dropConnectionsModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	dropBy := m.DropBy.ValueString()
	body := map[string]any{}
	dropByMap := map[string]any{"by": ""}

	switch dropBy {
	case "user_uuids":
		dropByMap["by"] = "userUuids"
		uuids := make([]string, 0, len(m.UserUUIDs.Elements()))
		for _, elem := range m.UserUUIDs.Elements() {
			uuids = append(uuids, elem.(types.String).ValueString())
		}
		if len(uuids) == 0 {
			diags.AddError("Invalid configuration", "user_uuids must not be empty when drop_by = user_uuids")
			return nil, diags
		}
		dropByMap["userUuids"] = uuids
	case "ip_addresses":
		dropByMap["by"] = "ipAddresses"
		ips := make([]string, 0, len(m.IPAddresses.Elements()))
		for _, elem := range m.IPAddresses.Elements() {
			ips = append(ips, elem.(types.String).ValueString())
		}
		if len(ips) == 0 {
			diags.AddError("Invalid configuration", "ip_addresses must not be empty when drop_by = ip_addresses")
			return nil, diags
		}
		dropByMap["ipAddresses"] = ips
	default:
		diags.AddError("Invalid drop_by", fmt.Sprintf("drop_by must be 'user_uuids' or 'ip_addresses', got %q", dropBy))
		return nil, diags
	}
	body["dropBy"] = dropByMap

	target := "allNodes"
	if !m.Target.IsNull() && !m.Target.IsUnknown() {
		switch m.Target.ValueString() {
		case "all_nodes", "":
			target = "allNodes"
		case "specific_nodes":
			target = "specificNodes"
		default:
			diags.AddError("Invalid target", fmt.Sprintf("target must be 'all_nodes' or 'specific_nodes', got %q", m.Target.ValueString()))
			return nil, diags
		}
	}

	targetNodesMap := map[string]any{"target": target}
	if target == "specificNodes" {
		nodeUUIDs := make([]string, 0, len(m.NodeUUIDs.Elements()))
		for _, elem := range m.NodeUUIDs.Elements() {
			nodeUUIDs = append(nodeUUIDs, elem.(types.String).ValueString())
		}
		if len(nodeUUIDs) == 0 {
			diags.AddError("Invalid configuration", "node_uuids must not be empty when target = specific_nodes")
			return nil, diags
		}
		targetNodesMap["nodeUuids"] = nodeUUIDs
	}
	body["targetNodes"] = targetNodesMap

	return body, diags
}

func computeDropConnectionsID(m *dropConnectionsModel) string {
	h := sha256.New()
	h.Write([]byte(m.DropBy.ValueString()))
	if !m.UserUUIDs.IsNull() {
		h.Write([]byte(m.UserUUIDs.String()))
	}
	if !m.IPAddresses.IsNull() {
		h.Write([]byte(m.IPAddresses.String()))
	}
	if !m.NodeUUIDs.IsNull() {
		h.Write([]byte(m.NodeUUIDs.String()))
	}
	if !m.Triggers.IsNull() {
		h.Write([]byte(m.Triggers.String()))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
