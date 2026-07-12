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

type subpageConfigResource struct {
	client *Client
}

type subpageConfigResourceModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Config types.String `tfsdk:"config"`
}

func NewSubpageConfigResource() resource.Resource {
	return &subpageConfigResource{}
}

func (r *subpageConfigResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_subpage_config"
}

func (r *subpageConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave subscription page config.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "UUID of the subscription page config.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the subscription page config (2-30 chars, letters/numbers/underscores/dashes/spaces).",
			},
			"config": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Subscription page config as a JSON string. Opaque to the provider — the panel manages the structure.",
			},
		},
	}
}

func (r *subpageConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subpageConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subpageConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create only accepts name — config is set via Update
	sc := &SubpageConfig{
		Name: plan.Name.ValueString(),
	}

	created, err := r.client.CreateSubpageConfig(ctx, sc)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create subscription page config", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// If config was specified in the plan, apply it via an Update call
	if !plan.Config.IsNull() && plan.Config.ValueString() != "" {
		var cfg any
		if err := json.Unmarshal([]byte(plan.Config.ValueString()), &cfg); err != nil {
			resp.Diagnostics.AddError("Invalid config JSON", err.Error())
			return
		}
		sc2 := &SubpageConfig{
			UUID:   created.UUID,
			Name:   plan.Name.ValueString(),
			Config: cfg,
		}
		updated, err := r.client.UpdateSubpageConfig(ctx, sc2)
		if err != nil {
			resp.Diagnostics.AddError("Failed to set config on create", err.Error())
			return
		}
		if updated.Config != nil {
			b, err := json.Marshal(updated.Config)
			if err != nil {
				resp.Diagnostics.AddError("Failed to marshal config", err.Error())
				return
			}
			plan.Config = types.StringValue(string(b))
		}
	} else if created.Config != nil {
		// Config not set in plan — populate from API default config
		b, err := json.Marshal(created.Config)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal config", err.Error())
			return
		}
		plan.Config = types.StringValue(string(b))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subpageConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subpageConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		return
	}

	sc, err := r.client.GetSubpageConfigByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "subscription page config not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read subscription page config", err.Error())
		return
	}

	state.UUID = types.StringValue(sc.UUID)
	state.Name = types.StringValue(sc.Name)
	if sc.Config != nil {
		b, err := json.Marshal(sc.Config)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal config", err.Error())
			return
		}
		state.Config = types.StringValue(string(b))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subpageConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan subpageConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sc := &SubpageConfig{
		UUID: plan.UUID.ValueString(),
		Name: plan.Name.ValueString(),
	}
	if !plan.Config.IsNull() && plan.Config.ValueString() != "" {
		var cfg any
		if err := json.Unmarshal([]byte(plan.Config.ValueString()), &cfg); err != nil {
			resp.Diagnostics.AddError("Invalid config JSON", err.Error())
			return
		}
		sc.Config = cfg
	}

	updated, err := r.client.UpdateSubpageConfig(ctx, sc)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update subscription page config", err.Error())
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

func (r *subpageConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subpageConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if err := r.client.DeleteSubpageConfig(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Failed to delete subscription page config", err.Error())
		return
	}
}

func (r *subpageConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
