package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type internalSquadResource struct{ client *Client }
type internalSquadModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Inbounds        types.Set    `tfsdk:"inbounds"`
	AccessibleNodes types.List   `tfsdk:"accessible_nodes"`
}

func NewInternalSquadResource() resource.Resource { return &internalSquadResource{} }

func (r *internalSquadResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_internal_squad"
}

func (r *internalSquadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave internal squad (group with inbound access control).",
		Attributes: map[string]schema.Attribute{
			"uuid":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":     schema.StringAttribute{Required: true, Description: "Squad name (2-30 chars)."},
			"inbounds": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Set of config profile inbound UUIDs."},
			"accessible_nodes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of accessible node UUIDs derived from the squad's inbound configuration (read-only).",
			},
		},
	}
}

func (r *internalSquadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *internalSquadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan internalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad := &InternalSquad{Name: plan.Name.ValueString()}
	if !plan.Inbounds.IsNull() && !plan.Inbounds.IsUnknown() {
		squad.Inbounds = make([]InternalSquadInboundRef, 0, len(plan.Inbounds.Elements()))
		for _, v := range plan.Inbounds.Elements() {
			squad.Inbounds = append(squad.Inbounds, InternalSquadInboundRef{UUID: v.(types.String).ValueString()})
		}
	}
	created, err := r.client.CreateInternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create internal squad", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	// Initialize accessible_nodes as empty list (computed, will be populated by Read)
	plan.AccessibleNodes, _ = types.ListValue(types.StringType, []attr.Value{})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *internalSquadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state internalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad, err := r.client.GetInternalSquadByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "internal squad not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read internal squad", err.Error())
		return
	}
	state.UUID = types.StringValue(squad.UUID)
	state.Name = types.StringValue(squad.Name)
	inboundElems := make([]attr.Value, 0, len(squad.Inbounds))
	for _, ib := range squad.Inbounds {
		inboundElems = append(inboundElems, types.StringValue(ib.UUID))
	}
	state.Inbounds, _ = types.SetValue(types.StringType, inboundElems)

	// Fetch accessible nodes (read-only derived data).
	accessible, err := r.client.GetInternalSquadAccessibleNodes(ctx, squad.UUID)
	elems := make([]attr.Value, 0)
	if err != nil {
		tflog.Warn(ctx, "failed to fetch accessible nodes", map[string]any{"uuid": squad.UUID, "err": err.Error()})
	} else {
		for _, n := range accessible.AccessibleNodes {
			elems = append(elems, types.StringValue(n.UUID))
		}
	}
	nodesList, _ := types.ListValue(types.StringType, elems)
	state.AccessibleNodes = nodesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *internalSquadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan internalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad := &InternalSquad{UUID: plan.UUID.ValueString(), Name: plan.Name.ValueString()}
	if !plan.Inbounds.IsNull() && !plan.Inbounds.IsUnknown() {
		squad.Inbounds = make([]InternalSquadInboundRef, 0, len(plan.Inbounds.Elements()))
		for _, v := range plan.Inbounds.Elements() {
			squad.Inbounds = append(squad.Inbounds, InternalSquadInboundRef{UUID: v.(types.String).ValueString()})
		}
	}
	updated, err := r.client.UpdateInternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update internal squad", err.Error())
		return
	}
	plan.UUID = types.StringValue(updated.UUID)
	plan.Name = types.StringValue(updated.Name)
	// Refresh accessible_nodes — inbounds may have changed, which affects
	// which nodes this squad can reach.
	accessible, err := r.client.GetInternalSquadAccessibleNodes(ctx, updated.UUID)
	elems := make([]attr.Value, 0)
	if err != nil {
		tflog.Warn(ctx, "failed to fetch accessible nodes after update", map[string]any{"uuid": updated.UUID, "err": err.Error()})
	} else {
		for _, n := range accessible.AccessibleNodes {
			elems = append(elems, types.StringValue(n.UUID))
		}
	}
	nodesList, _ := types.ListValue(types.StringType, elems)
	plan.AccessibleNodes = nodesList
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *internalSquadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state internalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInternalSquad(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete internal squad", err.Error())
	}
}

func (r *internalSquadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

var _ attr.Type = types.StringType
