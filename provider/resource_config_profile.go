package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	UUID     types.String `tfsdk:"uuid"`
	Name     types.String `tfsdk:"name"`
	Config   types.String `tfsdk:"config"`
	Inbounds types.List   `tfsdk:"inbounds"`
	Nodes    types.List   `tfsdk:"nodes"`
}

type configProfileInboundResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	ProfileUUID types.String `tfsdk:"profile_uuid"`
	Tag         types.String `tfsdk:"tag"`
	Type        types.String `tfsdk:"type"`
	Network     types.String `tfsdk:"network"`
	Security    types.String `tfsdk:"security"`
	Port        types.Int64  `tfsdk:"port"`
	RawInbound  types.String `tfsdk:"raw_inbound"`
}

type configProfileNodeResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	CountryCode types.String `tfsdk:"country_code"`
}

var configProfileInboundAttrTypes = map[string]attr.Type{
	"uuid":         types.StringType,
	"profile_uuid": types.StringType,
	"tag":          types.StringType,
	"type":         types.StringType,
	"network":      types.StringType,
	"security":     types.StringType,
	"port":         types.Int64Type,
	"raw_inbound":  types.StringType,
}

var configProfileNodeAttrTypes = map[string]attr.Type{
	"uuid":         types.StringType,
	"name":         types.StringType,
	"country_code": types.StringType,
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
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Xray configuration as JSON string. Opaque to the provider — the panel manages the structure.",
			},
			"inbounds": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Inbounds parsed by Remnawave from the Xray config. Their UUIDs are used by nodes, hosts, and squads.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"uuid":         schema.StringAttribute{Computed: true},
					"profile_uuid": schema.StringAttribute{Computed: true},
					"tag":          schema.StringAttribute{Computed: true},
					"type":         schema.StringAttribute{Computed: true},
					"network":      schema.StringAttribute{Computed: true},
					"security":     schema.StringAttribute{Computed: true},
					"port":         schema.Int64Attribute{Computed: true},
					"raw_inbound":  schema.StringAttribute{Computed: true, Sensitive: true, Description: "Raw inbound as normalized JSON."},
				}},
			},
			"nodes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Nodes currently assigned to this config profile.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"uuid":         schema.StringAttribute{Computed: true},
					"name":         schema.StringAttribute{Computed: true},
					"country_code": schema.StringAttribute{Computed: true},
				}},
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
		Name:   plan.Name.ValueString(),
		Config: map[string]any{}, // panel requires a config object, even if empty
	}
	if !plan.Config.IsNull() && plan.Config.ValueString() != "" {
		var cfg map[string]any
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

	if !configProfileToState(created, &plan, &resp.Diagnostics) {
		return
	}
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

	if !configProfileToState(profile, &state, &resp.Diagnostics) {
		return
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

	if !configProfileToState(updated, &plan, &resp.Diagnostics) {
		return
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

func configProfileToState(profile *ConfigProfile, state *configProfileResourceModel, diagnostics *diag.Diagnostics) bool {
	state.UUID = types.StringValue(profile.UUID)
	state.Name = types.StringValue(profile.Name)
	if profile.Config != nil {
		b, err := json.Marshal(profile.Config)
		if err != nil {
			diagnostics.AddError("Failed to marshal config", err.Error())
			return false
		}
		state.Config = types.StringValue(string(b))
	} else {
		state.Config = types.StringNull()
	}

	inbounds := make([]configProfileInboundResourceModel, 0, len(profile.Inbounds))
	for _, inbound := range profile.Inbounds {
		model := configProfileInboundResourceModel{
			UUID:        types.StringValue(inbound.UUID),
			ProfileUUID: types.StringValue(inbound.ProfileUUID),
			Tag:         types.StringValue(inbound.Tag),
			Type:        types.StringValue(inbound.Type),
			Network:     types.StringNull(),
			Security:    types.StringNull(),
			Port:        types.Int64Null(),
			RawInbound:  types.StringNull(),
		}
		if inbound.Network != nil {
			model.Network = types.StringValue(*inbound.Network)
		}
		if inbound.Security != nil {
			model.Security = types.StringValue(*inbound.Security)
		}
		if inbound.Port != nil {
			model.Port = types.Int64Value(int64(*inbound.Port))
		}
		if inbound.RawInbound != nil {
			b, err := json.Marshal(inbound.RawInbound)
			if err != nil {
				diagnostics.AddError("Failed to marshal raw inbound", err.Error())
				return false
			}
			model.RawInbound = types.StringValue(string(b))
		}
		inbounds = append(inbounds, model)
	}
	var conversionDiags diag.Diagnostics
	state.Inbounds, conversionDiags = types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: configProfileInboundAttrTypes}, inbounds)
	diagnostics.Append(conversionDiags...)
	if diagnostics.HasError() {
		return false
	}

	nodes := make([]configProfileNodeResourceModel, 0, len(profile.Nodes))
	for _, node := range profile.Nodes {
		nodes = append(nodes, configProfileNodeResourceModel{
			UUID:        types.StringValue(node.UUID),
			Name:        types.StringValue(node.Name),
			CountryCode: types.StringValue(node.CountryCode),
		})
	}
	state.Nodes, conversionDiags = types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: configProfileNodeAttrTypes}, nodes)
	diagnostics.Append(conversionDiags...)
	return !diagnostics.HasError()
}
