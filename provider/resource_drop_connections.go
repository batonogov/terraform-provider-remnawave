package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dropConnectionsResource is an imperative resource that triggers a
// "drop connections" action on the Remnawave IP Control module when applied.
// It uses a triggers map so that changes to trigger values cause the
// resource to be replaced (and the action re-executed).
type dropConnectionsResource struct {
	client *Client
}

type dropConnectionsModel struct {
	ID       types.String `tfsdk:"id"`
	UserUUID types.String `tfsdk:"user_uuid"`
	Triggers types.Map    `tfsdk:"triggers"`
}

func NewDropConnectionsResource() resource.Resource {
	return &dropConnectionsResource{}
}

func (r *dropConnectionsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_drop_connections"
}

func (r *dropConnectionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Drops all active connections for a user via the Remnawave IP Control module. " +
			"This is an imperative action resource: applying it sends a drop-connections request to the panel. " +
			"Use the optional triggers map to force re-execution when its values change.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Stable identifier derived from user_uuid and trigger values.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the user whose connections should be dropped.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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

	if err := r.client.DropConnections(ctx, plan.UserUUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to drop connections", err.Error())
		return
	}

	plan.ID = types.StringValue(computeDropConnectionsID(&plan))
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
	// Not practically importable, but provide a minimal implementation so the
	// resource satisfies the interface for tooling that expects it.
	resp.Diagnostics.AddError("Import not supported", "remnawave_drop_connections is an imperative action resource and cannot be imported.")
}

// ─── Helpers ───

func computeDropConnectionsID(m *dropConnectionsModel) string {
	h := sha256.New()
	h.Write([]byte(m.UserUUID.ValueString()))
	if !m.Triggers.IsNull() {
		// Triggers is a map; the framework ensures deterministic encoding via
		// Elements() iteration but we just need stability, so write the raw
		// string representation.
		h.Write([]byte(fmt.Sprintf("%v", m.Triggers.String())))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
