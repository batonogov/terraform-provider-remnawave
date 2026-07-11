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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type configProfileResource struct {
	client *Client
}

type configProfileResourceModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Config types.String `tfsdk:"config"`
}

func NewConfigProfileResource() resource.Resource {
	return &configProfileResource{}
}

func (r *configProfileResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_config_profile"
}

func (r *configProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave config profile (Xray configuration template).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "UUID of the config profile.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Profile name (2-30 chars, letters/numbers/underscores/dashes/spaces).",
			},
			"config": schema.StringAttribute{
				Optional:    true,
				Description: "Xray configuration as JSON string. Opaque to the provider — the panel manages the structure.",
			},
		},
	}
}

func (r *configProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile := &ConfigProfile{
		Name: plan.Name.ValueString(),
	}
	if !plan.Config.IsNull() && plan.Config.ValueString() != "" {
		var cfg any
		if err := json.Unmarshal([]byte(plan.Config.ValueString()), &cfg); err != nil {
			resp.Diagnostics.AddError("Invalid config JSON", err.Error())
			return
		}
		profile.Config = cfg
	}

	created, err := r.client.CreateConfigProfile(ctx, profile)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create config profile", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		return
	}

	profile, err := r.client.GetConfigProfileByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "config profile not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read config profile", err.Error())
		return
	}

	state.UUID = types.StringValue(profile.UUID)
	state.Name = types.StringValue(profile.Name)
	if profile.Config != nil {
		b, err := json.Marshal(profile.Config)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal config", err.Error())
			return
		}
		state.Config = types.StringValue(string(b))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan configProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile := &ConfigProfile{
		UUID: plan.UUID.ValueString(),
		Name: plan.Name.ValueString(),
	}
	if !plan.Config.IsNull() && plan.Config.ValueString() != "" {
		var cfg any
		if err := json.Unmarshal([]byte(plan.Config.ValueString()), &cfg); err != nil {
			resp.Diagnostics.AddError("Invalid config JSON", err.Error())
			return
		}
		profile.Config = cfg
	}

	updated, err := r.client.UpdateConfigProfile(ctx, profile)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update config profile", err.Error())
		return
	}

	plan.UUID = types.StringValue(updated.UUID)
	plan.Name = types.StringValue(updated.Name)
	if updated.Config != nil {
		b, err := json.Marshal(updated.Config)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal config", err.Error())
			return
		}
		plan.Config = types.StringValue(string(b))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if err := r.client.DeleteConfigProfile(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Failed to delete config profile", err.Error())
		return
	}
}

func (r *configProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
