package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type billingNodeResource struct{ client *Client }

type billingNodeModel struct {
	UUID          types.String `tfsdk:"uuid"`
	ProviderUUID  types.String `tfsdk:"provider_uuid"`
	NodeUUID      types.String `tfsdk:"node_uuid"`
	Name          types.String `tfsdk:"name"`
	NextBillingAt types.String `tfsdk:"next_billing_at"`
}

func NewBillingNodeResource() resource.Resource { return &billingNodeResource{} }

func (r *billingNodeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_billing_node"
}

func (r *billingNodeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave infra billing node.",
		Attributes: map[string]schema.Attribute{
			"uuid":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"provider_uuid":   schema.StringAttribute{Required: true, Description: "UUID of the infra provider."},
			"node_uuid":       schema.StringAttribute{Optional: true, Description: "UUID of the associated node (optional)."},
			"name":            schema.StringAttribute{Optional: true, Description: "Optional display name."},
			"next_billing_at": schema.StringAttribute{Required: true, Description: "Next billing date (ISO 8601 datetime)."},
		},
	}
}

func (r *billingNodeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *billingNodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan billingNodeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"providerUuid":  plan.ProviderUUID.ValueString(),
		"nextBillingAt": plan.NextBillingAt.ValueString(),
	}
	if !plan.NodeUUID.IsNull() && plan.NodeUUID.ValueString() != "" {
		body["nodeUuid"] = plan.NodeUUID.ValueString()
	}
	if !plan.Name.IsNull() && plan.Name.ValueString() != "" {
		body["name"] = plan.Name.ValueString()
	}

	out, err := r.client.CreateBillingNode(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create billing node", err.Error())
		return
	}

	// Find the created billing node by matching nextBillingAt
	var found *BillingNode
	target := plan.NextBillingAt.ValueString()
	for i := range out.BillingNodes {
		if out.BillingNodes[i].NextBillingAt == target {
			found = &out.BillingNodes[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Failed to create billing node", "created node not found in response")
		return
	}

	plan.UUID = types.StringValue(found.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billingNodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state billingNodeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetBillingNodes(ctx)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "billing node not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read billing nodes", err.Error())
		return
	}

	var found *BillingNode
	target := state.UUID.ValueString()
	for i := range out.BillingNodes {
		if out.BillingNodes[i].UUID == target {
			found = &out.BillingNodes[i]
			break
		}
	}
	if found == nil {
		tflog.Warn(ctx, "billing node not found in list", map[string]any{"uuid": target})
		resp.State.RemoveResource(ctx)
		return
	}

	state.UUID = types.StringValue(found.UUID)
	state.ProviderUUID = types.StringValue(found.ProviderUUID)
	state.NextBillingAt = types.StringValue(found.NextBillingAt)
	if found.NodeUUID != nil {
		state.NodeUUID = types.StringValue(*found.NodeUUID)
	} else {
		state.NodeUUID = types.StringNull()
	}
	if found.Name != nil {
		state.Name = types.StringValue(*found.Name)
	} else {
		state.Name = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *billingNodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan billingNodeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Batch update: send single-element uuids array with the new nextBillingAt
	body := map[string]any{
		"uuids":         []string{plan.UUID.ValueString()},
		"nextBillingAt": plan.NextBillingAt.ValueString(),
	}

	_, err := r.client.UpdateBillingNode(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update billing node", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billingNodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state billingNodeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBillingNode(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete billing node", err.Error())
	}
}

func (r *billingNodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
