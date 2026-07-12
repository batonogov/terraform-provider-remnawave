package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type panelSettingsResource struct{ client *Client }
type panelSettingsModel struct {
	ID                  types.String `tfsdk:"id"`
	BrandingTitle       types.String `tfsdk:"branding_title"`
	BrandingLogoURL     types.String `tfsdk:"branding_logo_url"`
	PasswordAuthEnabled types.Bool   `tfsdk:"password_auth_enabled"`
	PasskeySettings     types.String `tfsdk:"passkey_settings"`
	OAuth2Settings      types.String `tfsdk:"oauth2_settings"`
}

func NewPanelSettingsResource() resource.Resource { return &panelSettingsResource{} }

func (r *panelSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_panel_settings"
}

func (r *panelSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Remnawave panel settings (singleton: branding, auth).",
		Attributes: map[string]schema.Attribute{
			"id":                    schema.StringAttribute{Computed: true, Description: "Always 'settings'."},
			"branding_title":        schema.StringAttribute{Optional: true, Computed: true, Description: "Panel branding title."},
			"branding_logo_url":     schema.StringAttribute{Optional: true, Computed: true, Description: "Panel branding logo URL."},
			"password_auth_enabled": schema.BoolAttribute{Optional: true, Computed: true, Description: "Enable password auth."},
			"passkey_settings":      schema.StringAttribute{Optional: true, Computed: true, Description: "Passkey/WebAuthn settings as JSON string."},
			"oauth2_settings":       schema.StringAttribute{Optional: true, Computed: true, Description: "OAuth2 provider settings as JSON string."},
		},
	}
}

func (r *panelSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *panelSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan panelSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings := planToPanelSettings(&plan)
	updated, err := r.client.UpdatePanelSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to set panel settings", err.Error())
		return
	}
	panelSettingsToPlan(updated, &plan)
	plan.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *panelSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state panelSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.GetPanelSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read panel settings", err.Error())
		return
	}
	panelSettingsToPlan(settings, &state)
	state.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *panelSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan panelSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings := planToPanelSettings(&plan)
	updated, err := r.client.UpdatePanelSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update panel settings", err.Error())
		return
	}
	panelSettingsToPlan(updated, &plan)
	plan.ID = types.StringValue("settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *panelSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton — no-op
}

func (r *panelSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue("settings"))...)
}

func planToPanelSettings(p *panelSettingsModel) *PanelSettings {
	s := &PanelSettings{}
	if !p.BrandingTitle.IsNull() || !p.BrandingLogoURL.IsNull() {
		s.BrandingSettings = &BrandingSettings{}
		if !p.BrandingTitle.IsNull() {
			t := p.BrandingTitle.ValueString()
			s.BrandingSettings.Title = &t
		}
		if !p.BrandingLogoURL.IsNull() {
			u := p.BrandingLogoURL.ValueString()
			s.BrandingSettings.LogoURL = &u
		}
	}
	if !p.PasswordAuthEnabled.IsNull() {
		s.PasswordSettings = &PasswordAuthSettings{}
		e := p.PasswordAuthEnabled.ValueBool()
		s.PasswordSettings.Enabled = &e
	}
	if !p.PasskeySettings.IsNull() {
		var cfg any
		if err := json.Unmarshal([]byte(p.PasskeySettings.ValueString()), &cfg); err == nil {
			s.PasskeySettings = cfg
		}
	}
	if !p.OAuth2Settings.IsNull() {
		var cfg any
		if err := json.Unmarshal([]byte(p.OAuth2Settings.ValueString()), &cfg); err == nil {
			s.OAuth2Settings = cfg
		}
	}
	return s
}

func panelSettingsToPlan(s *PanelSettings, p *panelSettingsModel) {
	if s.BrandingSettings != nil {
		if s.BrandingSettings.Title != nil {
			p.BrandingTitle = types.StringValue(*s.BrandingSettings.Title)
		}
		if s.BrandingSettings.LogoURL != nil {
			p.BrandingLogoURL = types.StringValue(*s.BrandingSettings.LogoURL)
		}
	}
	if s.PasswordSettings != nil && s.PasswordSettings.Enabled != nil {
		p.PasswordAuthEnabled = types.BoolValue(*s.PasswordSettings.Enabled)
	}
	if s.PasskeySettings != nil {
		if b, err := json.Marshal(s.PasskeySettings); err == nil {
			p.PasskeySettings = types.StringValue(string(b))
		}
	}
	if s.OAuth2Settings != nil {
		if b, err := json.Marshal(s.OAuth2Settings); err == nil {
			p.OAuth2Settings = types.StringValue(string(b))
		}
	}
}
