package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nodeMetadataResource struct{ client *Client }

type nodeMetadataModel struct {
	UUID     types.String `tfsdk:"uuid"`
	NodeUUID types.String `tfsdk:"node_uuid"`
	Metadata types.String `tfsdk:"metadata"`
}

func NewNodeMetadataResource() resource.Resource { return &nodeMetadataResource{} }

func (r *nodeMetadataResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node_metadata"
}

func (r *nodeMetadataResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages free-form metadata for a Remnawave node. Metadata is upserted (PUT); there is no DELETE — clear it by setting metadata to {}.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the node this metadata belongs to (equals node_uuid).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the node whose metadata is managed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metadata": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Free-form metadata as a JSON object string, e.g. jsonencode({ location = \"us-east\" }).",
			},
		},
	}
}

func (r *nodeMetadataResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodeMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nodeMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(plan.Metadata.ValueString()), &meta); err != nil {
		resp.Diagnostics.AddError("Invalid metadata JSON", err.Error())
		return
	}

	out, err := r.client.UpsertNodeMetadata(ctx, plan.NodeUUID.ValueString(), meta)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upsert node metadata", err.Error())
		return
	}

	plan.UUID = plan.NodeUUID
	plan.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nodeMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.NodeUUID.ValueString()
	if uuid == "" {
		uuid = state.UUID.ValueString()
	}
	if uuid == "" {
		return
	}

	out, err := r.client.GetNodeMetadata(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read node metadata", err.Error())
		return
	}

	state.UUID = types.StringValue(uuid)
	state.NodeUUID = types.StringValue(uuid)
	state.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nodeMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nodeMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(plan.Metadata.ValueString()), &meta); err != nil {
		resp.Diagnostics.AddError("Invalid metadata JSON", err.Error())
		return
	}

	out, err := r.client.UpsertNodeMetadata(ctx, plan.NodeUUID.ValueString(), meta)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update node metadata", err.Error())
		return
	}

	plan.UUID = plan.NodeUUID
	plan.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Metadata has no DELETE endpoint — upsert empty object to clear.
	var state nodeMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.NodeUUID.ValueString()
	if uuid == "" {
		uuid = state.UUID.ValueString()
	}
	if uuid == "" {
		return
	}

	if _, err := r.client.UpsertNodeMetadata(ctx, uuid, map[string]any{}); err != nil {
		resp.Diagnostics.AddError("Failed to clear node metadata", err.Error())
		return
	}
}

func (r *nodeMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("node_uuid"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
