package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type sharedListResource struct{ client *Client }

// sharedListNamePattern mirrors the Remnawave 3.4 backend name grammar:
// letters, numbers, underscores and dashes, optionally in slash-separated
// segments. Pre-3.4 panels accept the same shape minus the slashes and
// reject the rest server-side.
var sharedListNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$`)

type sharedListResourceModel struct {
	Name   types.String `tfsdk:"name"`
	Config types.String `tfsdk:"config"`
}

func NewSharedListResource() resource.Resource { return &sharedListResource{} }

func (r *sharedListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_shared_list"
}

func (r *sharedListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a global Remnawave 3.3+ node-plugin shared list.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Shared-list name without the ext: prefix (2-255 letters, numbers, underscores, dashes, and single slashes between segments; slashes require Remnawave 3.4+).",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 255),
					// Matches the Remnawave 3.4 backend rule; older panels
					// reject slash-separated names server-side.
					stringvalidator.RegexMatches(sharedListNamePattern, "name may contain only letters, numbers, underscores, dashes, and slash-separated segments"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Shared-list configuration as JSON: an ipList with IP/CIDR items or an asList with numeric ASN items.",
			},
		},
	}
}

func (r *sharedListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sharedListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if err := requireBackend3_3(ctx, r.client, "shared lists"); err != nil {
		resp.Diagnostics.AddError("Unsupported shared list", err.Error())
		return
	}
	var plan sharedListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := sharedListFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid shared list", err.Error())
		return
	}
	created, err := r.client.CreateSharedList(ctx, list)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create shared list", err.Error())
		return
	}
	if err := sharedListToModel(created, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to decode shared list", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if err := requireBackend3_3(ctx, r.client, "shared lists"); err != nil {
		resp.Diagnostics.AddError("Unsupported shared list", err.Error())
		return
	}
	var state sharedListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.GetSharedListByName(ctx, state.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "shared list not found", map[string]any{"name": state.Name.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read shared list", err.Error())
		return
	}
	if err := sharedListToModel(list, &state); err != nil {
		resp.Diagnostics.AddError("Failed to decode shared list", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sharedListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if err := requireBackend3_3(ctx, r.client, "shared lists"); err != nil {
		resp.Diagnostics.AddError("Unsupported shared list", err.Error())
		return
	}
	var plan sharedListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := sharedListFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid shared list", err.Error())
		return
	}
	updated, err := r.client.UpdateSharedList(ctx, list)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update shared list", err.Error())
		return
	}
	if err := sharedListToModel(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to decode shared list", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := requireBackend3_3(ctx, r.client, "shared lists"); err != nil {
		resp.Diagnostics.AddError("Unsupported shared list", err.Error())
		return
	}
	var state sharedListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSharedList(ctx, state.Name.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete shared list", err.Error())
	}
}

func (r *sharedListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(req.ID))...)
}

func sharedListFromModel(model *sharedListResourceModel) (*SharedList, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(model.Config.ValueString()), &config); err != nil {
		return nil, fmt.Errorf("config must be a JSON object: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	return &SharedList{Name: model.Name.ValueString(), Config: config}, nil
}

func sharedListToModel(list *SharedList, model *sharedListResourceModel) error {
	config, err := json.Marshal(list.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	model.Name = types.StringValue(list.Name)
	model.Config = types.StringValue(string(config))
	return nil
}
