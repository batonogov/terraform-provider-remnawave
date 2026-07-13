package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type subscriptionSettingsResource struct {
	client *Client
}

type subscriptionSettingsModel struct {
	ID                          types.String `tfsdk:"id"`
	ProfileTitle                types.String `tfsdk:"profile_title"`
	SupportLink                 types.String `tfsdk:"support_link"`
	ProfileUpdateInterval       types.Int64  `tfsdk:"profile_update_interval"`
	IsProfileWebpageURLEnabled  types.Bool   `tfsdk:"is_profile_webpage_url_enabled"`
	ServeJsonAtBaseSubscription types.Bool   `tfsdk:"serve_json_at_base_subscription"`
	IsShowCustomRemarks         types.Bool   `tfsdk:"is_show_custom_remarks"`
	HappAnnounce                types.String `tfsdk:"happ_announce"`
	HappRouting                 types.String `tfsdk:"happ_routing"`
	RandomizeHosts              types.Bool   `tfsdk:"randomize_hosts"`
	CustomRemarks               types.String `tfsdk:"custom_remarks"`
	CustomResponseHeaders       types.String `tfsdk:"custom_response_headers"`
	ResponseRules               types.String `tfsdk:"response_rules"`
	HwidSettings                types.String `tfsdk:"hwid_settings"`
}

func NewSubscriptionSettingsResource() resource.Resource {
	return &subscriptionSettingsResource{}
}

func (r *subscriptionSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_subscription_settings"
}

func (r *subscriptionSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Remnawave subscription settings (singleton). Delete only removes from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Always 'settings' — this is a singleton resource.",
			},
			"profile_title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Subscription profile title shown in VPN clients.",
			},
			"support_link": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Support link shown in subscription page.",
			},
			"profile_update_interval": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Subscription update interval in minutes.",
			},
			"is_profile_webpage_url_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable profile webpage URL.",
			},
			"serve_json_at_base_subscription": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Serve JSON at base subscription URL.",
			},
			"is_show_custom_remarks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Show custom remarks for users.",
			},
			"happ_announce": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Happ announce message (max 200 chars).",
			},
			"happ_routing": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Happ routing config.",
			},
			"randomize_hosts": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Randomize host order in subscription.",
			},
			"custom_remarks": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Custom user-state remarks as JSON.",
			},
			"custom_response_headers": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Custom subscription response headers as JSON object.",
			},
			"response_rules": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Subscription response-rules configuration as JSON.",
			},
			"hwid_settings": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "HWID enforcement settings as JSON.",
			},
		},
	}
}

func (r *subscriptionSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subscriptionSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subscriptionSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GET current settings to obtain UUID (required for PATCH)
	current, err := r.client.GetSubscriptionSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read current subscription settings", err.Error())
		return
	}

	settings := planToSubscriptionSettings(&plan)
	settings.UUID = current.UUID
	updated, err := r.client.UpdateSubscriptionSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to set subscription settings", err.Error())
		return
	}

	subscriptionSettingsToPlan(updated, &plan)
	plan.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subscriptionSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetSubscriptionSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read subscription settings", err.Error())
		return
	}

	subscriptionSettingsToPlan(settings, &state)
	state.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriptionSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan subscriptionSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GET current to obtain UUID (required for PATCH)
	current, err := r.client.GetSubscriptionSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read current subscription settings", err.Error())
		return
	}

	settings := planToSubscriptionSettings(&plan)
	settings.UUID = current.UUID
	updated, err := r.client.UpdateSubscriptionSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update subscription settings", err.Error())
		return
	}

	subscriptionSettingsToPlan(updated, &plan)
	plan.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton — delete only removes from TF state, does not reset panel settings.
}

func (r *subscriptionSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// subscription_settings is a singleton — id is always "settings".
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue("settings"))...)
}

// ─── Conversions ───

func planToSubscriptionSettings(p *subscriptionSettingsModel) *SubscriptionSettings {
	s := &SubscriptionSettings{}
	if !p.ProfileTitle.IsNull() {
		v := p.ProfileTitle.ValueString()
		s.ProfileTitle = &v
	}
	if !p.SupportLink.IsNull() {
		v := p.SupportLink.ValueString()
		s.SupportLink = &v
	}
	if !p.ProfileUpdateInterval.IsNull() {
		v := int(p.ProfileUpdateInterval.ValueInt64())
		s.ProfileUpdateInterval = &v
	}
	if !p.IsProfileWebpageURLEnabled.IsNull() {
		v := p.IsProfileWebpageURLEnabled.ValueBool()
		s.IsProfileWebpageURLEnabled = &v
	}
	if !p.ServeJsonAtBaseSubscription.IsNull() {
		v := p.ServeJsonAtBaseSubscription.ValueBool()
		s.ServeJsonAtBaseSubscription = &v
	}
	if !p.IsShowCustomRemarks.IsNull() {
		v := p.IsShowCustomRemarks.ValueBool()
		s.IsShowCustomRemarks = &v
	}
	if !p.HappAnnounce.IsNull() {
		v := p.HappAnnounce.ValueString()
		s.HappAnnounce = &v
	}
	if !p.HappRouting.IsNull() {
		v := p.HappRouting.ValueString()
		s.HappRouting = &v
	}
	if !p.RandomizeHosts.IsNull() {
		v := p.RandomizeHosts.ValueBool()
		s.RandomizeHosts = &v
	}
	if !p.CustomRemarks.IsNull() && !p.CustomRemarks.IsUnknown() {
		s.CustomRemarks = json.RawMessage(p.CustomRemarks.ValueString())
	}
	if !p.CustomResponseHeaders.IsNull() && !p.CustomResponseHeaders.IsUnknown() {
		s.CustomResponseHeaders = json.RawMessage(p.CustomResponseHeaders.ValueString())
	}
	if !p.ResponseRules.IsNull() && !p.ResponseRules.IsUnknown() {
		s.ResponseRules = json.RawMessage(p.ResponseRules.ValueString())
	}
	if !p.HwidSettings.IsNull() && !p.HwidSettings.IsUnknown() {
		s.HwidSettings = json.RawMessage(p.HwidSettings.ValueString())
	}
	return s
}

func subscriptionSettingsToPlan(s *SubscriptionSettings, p *subscriptionSettingsModel) {
	if s.ProfileTitle != nil {
		p.ProfileTitle = types.StringValue(*s.ProfileTitle)
	} else {
		p.ProfileTitle = types.StringNull()
	}
	if s.SupportLink != nil {
		p.SupportLink = types.StringValue(*s.SupportLink)
	} else {
		p.SupportLink = types.StringNull()
	}
	if s.ProfileUpdateInterval != nil {
		p.ProfileUpdateInterval = types.Int64Value(int64(*s.ProfileUpdateInterval))
	} else {
		p.ProfileUpdateInterval = types.Int64Null()
	}
	if s.IsProfileWebpageURLEnabled != nil {
		p.IsProfileWebpageURLEnabled = types.BoolValue(*s.IsProfileWebpageURLEnabled)
	} else {
		p.IsProfileWebpageURLEnabled = types.BoolNull()
	}
	if s.ServeJsonAtBaseSubscription != nil {
		p.ServeJsonAtBaseSubscription = types.BoolValue(*s.ServeJsonAtBaseSubscription)
	} else {
		p.ServeJsonAtBaseSubscription = types.BoolNull()
	}
	if s.IsShowCustomRemarks != nil {
		p.IsShowCustomRemarks = types.BoolValue(*s.IsShowCustomRemarks)
	} else {
		p.IsShowCustomRemarks = types.BoolNull()
	}
	if s.HappAnnounce != nil {
		p.HappAnnounce = types.StringValue(*s.HappAnnounce)
	} else {
		p.HappAnnounce = types.StringNull()
	}
	if s.HappRouting != nil {
		p.HappRouting = types.StringValue(*s.HappRouting)
	} else {
		p.HappRouting = types.StringNull()
	}
	if s.RandomizeHosts != nil {
		p.RandomizeHosts = types.BoolValue(*s.RandomizeHosts)
	} else {
		p.RandomizeHosts = types.BoolNull()
	}
	p.CustomRemarks = rawJSONToString(s.CustomRemarks)
	p.CustomResponseHeaders = rawJSONToString(s.CustomResponseHeaders)
	p.ResponseRules = rawJSONToString(s.ResponseRules)
	p.HwidSettings = rawJSONToString(s.HwidSettings)
}

func rawJSONToString(value json.RawMessage) types.String {
	if len(value) == 0 || string(value) == "null" {
		return types.StringNull()
	}
	var normalized any
	if err := json.Unmarshal(value, &normalized); err != nil {
		return types.StringValue(string(value))
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return types.StringValue(string(value))
	}
	return types.StringValue(string(b))
}
