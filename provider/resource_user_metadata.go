package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type userMetadataResource struct{ client *Client }

type userMetadataModel struct {
	UUID     types.String `tfsdk:"uuid"`
	UserUUID types.String `tfsdk:"user_uuid"`
	Metadata types.String `tfsdk:"metadata"`
}

func NewUserMetadataResource() resource.Resource { return &userMetadataResource{} }

func (r *userMetadataResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_user_metadata"
}

func (r *userMetadataResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages free-form metadata for a Remnawave user. Metadata is upserted (PUT); there is no DELETE — clear it by setting metadata to {}.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the user this metadata belongs to (equals user_uuid).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the user whose metadata is managed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metadata": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Free-form metadata as a JSON object string, e.g. jsonencode({ department = \"engineering\" }).",
			},
		},
	}
}

func (r *userMetadataResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(plan.Metadata.ValueString()), &meta); err != nil {
		resp.Diagnostics.AddError("Invalid metadata JSON", err.Error())
		return
	}

	out, err := r.client.UpsertUserMetadata(ctx, plan.UserUUID.ValueString(), meta)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upsert user metadata", err.Error())
		return
	}

	plan.UUID = plan.UserUUID
	plan.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UserUUID.ValueString()
	if uuid == "" {
		uuid = state.UUID.ValueString()
	}
	if uuid == "" {
		return
	}

	out, err := r.client.GetUserMetadata(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read user metadata", err.Error())
		return
	}

	state.UUID = types.StringValue(uuid)
	state.UserUUID = types.StringValue(uuid)
	state.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(plan.Metadata.ValueString()), &meta); err != nil {
		resp.Diagnostics.AddError("Invalid metadata JSON", err.Error())
		return
	}

	out, err := r.client.UpsertUserMetadata(ctx, plan.UserUUID.ValueString(), meta)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user metadata", err.Error())
		return
	}

	plan.UUID = plan.UserUUID
	plan.Metadata = types.StringValue(metadataToJSON(out, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Metadata has no DELETE endpoint — upsert empty object to clear.
	var state userMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UserUUID.ValueString()
	if uuid == "" {
		uuid = state.UUID.ValueString()
	}
	if uuid == "" {
		return
	}

	if _, err := r.client.UpsertUserMetadata(ctx, uuid, map[string]any{}); err != nil {
		resp.Diagnostics.AddError("Failed to clear user metadata", err.Error())
		return
	}
}

func (r *userMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_uuid"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

// metadataToJSON extracts the "metadata" field from the API response and
// marshals it to a canonical JSON string. Falls back to the raw map.
func metadataToJSON(resp map[string]any, diags *diag.Diagnostics) string {
	if raw, ok := resp["metadata"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			return string(b)
		}
	}
	// Fallback: try marshalling the whole response
	if b, err := json.Marshal(resp); err == nil {
		return string(b)
	}
	diags.AddError("Failed to marshal metadata response", "")
	return "{}"
}
