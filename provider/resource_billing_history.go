package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type billingHistoryResource struct{ client *Client }

type billingHistoryModel struct {
	UUID         types.String  `tfsdk:"uuid"`
	ProviderUUID types.String  `tfsdk:"provider_uuid"`
	Amount       types.Float64 `tfsdk:"amount"`
	BilledAt     types.String  `tfsdk:"billed_at"`
}

func NewBillingHistoryResource() resource.Resource { return &billingHistoryResource{} }

func (r *billingHistoryResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_billing_history"
}

func (r *billingHistoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave infra billing history record. Records cannot be updated — only created and deleted.",
		Attributes: map[string]schema.Attribute{
			"uuid":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"provider_uuid": schema.StringAttribute{Required: true, Description: "UUID of the infra provider."},
			"amount":        schema.Float64Attribute{Required: true, PlanModifiers: []planmodifier.Float64{float64planmodifier.RequiresReplace()}, Description: "Billing amount."},
			"billed_at":     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Billing date (ISO 8601 datetime)."},
		},
	}
}

func (r *billingHistoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *billingHistoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan billingHistoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"providerUuid": plan.ProviderUUID.ValueString(),
		"amount":       plan.Amount.ValueFloat64(),
		"billedAt":     plan.BilledAt.ValueString(),
	}

	out, err := r.client.CreateBillingHistory(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create billing history record", err.Error())
		return
	}

	// Find the created record by matching amount + billedAt
	targetAmount := plan.Amount.ValueFloat64()
	targetBilledAt := plan.BilledAt.ValueString()
	var found *BillingHistoryRecord
	for i := range out.Records {
		if out.Records[i].Amount == targetAmount && out.Records[i].BilledAt == targetBilledAt {
			found = &out.Records[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Failed to create billing history record", "created record not found in response")
		return
	}

	plan.UUID = types.StringValue(found.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billingHistoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state billingHistoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetBillingHistory(ctx)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "billing history record not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read billing history", err.Error())
		return
	}

	var found *BillingHistoryRecord
	target := state.UUID.ValueString()
	for i := range out.Records {
		if out.Records[i].UUID == target {
			found = &out.Records[i]
			break
		}
	}
	if found == nil {
		tflog.Warn(ctx, "billing history record not found in list", map[string]any{"uuid": target})
		resp.State.RemoveResource(ctx)
		return
	}

	state.UUID = types.StringValue(found.UUID)
	state.ProviderUUID = types.StringValue(found.ProviderUUID)
	state.Amount = types.Float64Value(found.Amount)
	state.BilledAt = types.StringValue(found.BilledAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *billingHistoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update API — all fields use RequiresReplace, so Terraform will
	// destroy and recreate. This method should never be called.
	resp.Diagnostics.AddError("Update not supported", "billing history records cannot be updated, only deleted and recreated")
}

func (r *billingHistoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state billingHistoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBillingHistory(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete billing history record", err.Error())
	}
}

func (r *billingHistoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
