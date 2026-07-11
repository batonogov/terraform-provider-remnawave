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

type externalSquadResource struct {
	client *Client
}

type externalSquadModel struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
}

func NewExternalSquadResource() resource.Resource {
	return &externalSquadResource{}
}

func (r *externalSquadResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_external_squad"
}

func (r *externalSquadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave external squad.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Squad name (2-30 chars).",
			},
		},
	}
}

func (r *externalSquadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *externalSquadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan externalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	squad := &ExternalSquad{Name: plan.Name.ValueString()}
	created, err := r.client.CreateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create external squad", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *externalSquadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state externalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	squad, err := r.client.GetExternalSquadByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "external squad not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read external squad", err.Error())
		return
	}

	state.UUID = types.StringValue(squad.UUID)
	state.Name = types.StringValue(squad.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *externalSquadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan externalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	squad := &ExternalSquad{
		UUID: plan.UUID.ValueString(),
		Name: plan.Name.ValueString(),
	}
	updated, err := r.client.UpdateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external squad", err.Error())
		return
	}

	plan.UUID = types.StringValue(updated.UUID)
	plan.Name = types.StringValue(updated.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *externalSquadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state externalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteExternalSquad(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete external squad", err.Error())
		return
	}
}

func (r *externalSquadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
