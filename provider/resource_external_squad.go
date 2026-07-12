package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type externalSquadResource struct{ client *Client }

type externalSquadTemplateModel struct {
	TemplateUUID types.String `tfsdk:"template_uuid"`
	TemplateType types.String `tfsdk:"template_type"`
}

type externalSquadModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Templates            types.List   `tfsdk:"templates"`
	SubscriptionSettings types.String `tfsdk:"subscription_settings"`
	HostOverrides        types.String `tfsdk:"host_overrides"`
	ResponseHeaders      types.Map    `tfsdk:"response_headers"`
	HwidSettings         types.String `tfsdk:"hwid_settings"`
	CustomRemarks        types.String `tfsdk:"custom_remarks"`
	SubpageConfigUUID    types.String `tfsdk:"subpage_config_uuid"`
}

func NewExternalSquadResource() resource.Resource { return &externalSquadResource{} }

func (r *externalSquadResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_external_squad"
}

func (r *externalSquadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave external squad.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Required: true, Description: "Squad name (2-30 chars)."},
			"templates": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Subscription templates assigned to this squad.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"template_uuid": schema.StringAttribute{Required: true, Description: "UUID of the subscription template."},
						"template_type": schema.StringAttribute{Required: true, Description: "Type: VLESS, TROJAN, SHADOWSOCKS, etc."},
					},
				},
			},
			"subscription_settings": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Subscription settings as JSON string.",
			},
			"host_overrides": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Host overrides as JSON string.",
			},
			"response_headers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Custom HTTP response headers.",
			},
			"hwid_settings": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "HWID settings as JSON string.",
			},
			"custom_remarks": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Custom remarks as JSON string.",
			},
			"subpage_config_uuid": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "UUID of the subscription page config.",
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

	// Create only accepts name
	created, err := r.client.CreateExternalSquad(ctx, &ExternalSquad{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create external squad", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)

	// After create, send an update with all the extended fields
	squad := planToExternalSquad(&plan, &resp.Diagnostics)
	squad.UUID = created.UUID
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external squad after create", err.Error())
		return
	}
	externalSquadToState(updated, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *externalSquadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state externalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad, err := r.client.GetExternalSquadByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "external squad not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read external squad", err.Error())
		return
	}
	externalSquadToState(squad, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *externalSquadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan externalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad := planToExternalSquad(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external squad", err.Error())
		return
	}
	externalSquadToState(updated, &plan, &resp.Diagnostics)
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
	}
}

func (r *externalSquadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

// ─── Conversions ───

func planToExternalSquad(p *externalSquadModel, diags *diag.Diagnostics) *ExternalSquad {
	squad := &ExternalSquad{
		UUID: p.UUID.ValueString(),
		Name: p.Name.ValueString(),
	}

	// Templates
	if !p.Templates.IsNull() && !p.Templates.IsUnknown() {
		var tpls []externalSquadTemplateModel
		diags.Append(p.Templates.ElementsAs(context.Background(), &tpls, false)...)
		if diags.HasError() {
			return squad
		}
		for _, t := range tpls {
			squad.Templates = append(squad.Templates, ExternalSquadTemplate{
				TemplateUUID: t.TemplateUUID.ValueString(),
				TemplateType: t.TemplateType.ValueString(),
			})
		}
	}

	// JSON fields: subscription_settings, host_overrides, hwid_settings, custom_remarks
	squad.SubscriptionSettings = jsonStringToAny(p.SubscriptionSettings, diags)
	squad.HostOverrides = jsonStringToAny(p.HostOverrides, diags)
	squad.HwidSettings = jsonStringToAny(p.HwidSettings, diags)
	squad.CustomRemarks = jsonStringToAny(p.CustomRemarks, diags)

	// Response headers (map[string]string)
	if !p.ResponseHeaders.IsNull() && !p.ResponseHeaders.IsUnknown() {
		m := make(map[string]string)
		for k, v := range p.ResponseHeaders.Elements() {
			m[k] = v.(basetypes.StringValue).ValueString()
		}
		squad.ResponseHeaders = m
	}

	// Subpage config UUID (nullable)
	if !p.SubpageConfigUUID.IsNull() && !p.SubpageConfigUUID.IsUnknown() {
		val := p.SubpageConfigUUID.ValueString()
		if val != "" {
			squad.SubpageConfigUUID = &val
		}
	}

	return squad
}

// jsonStringToAny unmarshals a JSON string into an `any` value for the API.
// Returns nil for null/unknown/empty strings.
func jsonStringToAny(s types.String, diags *diag.Diagnostics) any {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	raw := s.ValueString()
	if raw == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		diags.AddError("Invalid JSON", fmt.Sprintf("failed to parse JSON: %s\nValue: %s", err.Error(), raw))
		return nil
	}
	return v
}

func externalSquadToState(s *ExternalSquad, p *externalSquadModel, diags *diag.Diagnostics) {
	p.UUID = types.StringValue(s.UUID)
	p.Name = types.StringValue(s.Name)

	// Templates
	if len(s.Templates) > 0 {
		tpls := make([]externalSquadTemplateModel, 0, len(s.Templates))
		for _, t := range s.Templates {
			tpls = append(tpls, externalSquadTemplateModel{
				TemplateUUID: types.StringValue(t.TemplateUUID),
				TemplateType: types.StringValue(t.TemplateType),
			})
		}
		listVal, d := types.ListValueFrom(context.Background(), types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"template_uuid": types.StringType,
				"template_type": types.StringType,
			},
		}, tpls)
		diags.Append(d...)
		p.Templates = listVal
	} else {
		p.Templates = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"template_uuid": types.StringType,
				"template_type": types.StringType,
			},
		})
	}

	// JSON fields
	p.SubscriptionSettings = anyToJSONString(s.SubscriptionSettings, diags)
	p.HostOverrides = anyToJSONString(s.HostOverrides, diags)
	p.HwidSettings = anyToJSONString(s.HwidSettings, diags)
	p.CustomRemarks = anyToJSONString(s.CustomRemarks, diags)

	// Response headers
	if len(s.ResponseHeaders) > 0 {
		elems := make(map[string]attr.Value, len(s.ResponseHeaders))
		for k, v := range s.ResponseHeaders {
			elems[k] = types.StringValue(v)
		}
		mapVal, d := types.MapValue(types.StringType, elems)
		diags.Append(d...)
		p.ResponseHeaders = mapVal
	} else {
		p.ResponseHeaders = types.MapNull(types.StringType)
	}

	// Subpage config UUID
	if s.SubpageConfigUUID != nil {
		p.SubpageConfigUUID = types.StringValue(*s.SubpageConfigUUID)
	} else {
		p.SubpageConfigUUID = types.StringNull()
	}
}

// anyToJSONString marshals an `any` value to a JSON string for Terraform state.
func anyToJSONString(v any, diags *diag.Diagnostics) types.String {
	if v == nil {
		return types.StringNull()
	}
	raw, err := json.Marshal(v)
	if err != nil {
		diags.AddError("Failed to marshal JSON", err.Error())
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}
