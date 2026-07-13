package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type externalSquadResource struct{ client *Client }
type externalSquadModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Templates            types.String `tfsdk:"templates"`
	SubscriptionSettings types.String `tfsdk:"subscription_settings"`
	HostOverrides        types.String `tfsdk:"host_overrides"`
	ResponseHeaders      types.String `tfsdk:"response_headers"`
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
			"uuid":                  schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                  schema.StringAttribute{Required: true, Description: "Squad name (2-30 chars)."},
			"templates":             schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Template assignments as JSON array."},
			"subscription_settings": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad-specific subscription settings as JSON."},
			"host_overrides":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad host overrides as JSON."},
			"response_headers":      schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad response headers as JSON object."},
			"hwid_settings":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad HWID settings as JSON."},
			"custom_remarks":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad custom remarks as JSON."},
			"subpage_config_uuid":   schema.StringAttribute{Optional: true, Computed: true, Description: "Subscription page config UUID assigned to the squad."},
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
	update, err := externalSquadFromPlan(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	created, err := r.client.CreateExternalSquad(ctx, &ExternalSquad{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create external squad", err.Error())
		return
	}
	update.UUID = created.UUID
	updated, err := r.client.UpdateExternalSquad(ctx, update)
	if err != nil {
		detail := err.Error()
		if cleanupErr := r.client.DeleteExternalSquad(ctx, created.UUID); cleanupErr != nil {
			detail += fmt.Sprintf("; additionally failed to delete partially created squad %s: %v", created.UUID, cleanupErr)
		}
		resp.Diagnostics.AddError("Failed to set external squad configuration", detail)
		return
	}
	externalSquadToPlan(updated, &plan)
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
	externalSquadToPlan(squad, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *externalSquadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan externalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	squad, err := externalSquadFromPlan(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	squad.UUID = plan.UUID.ValueString()
	updated, err := r.client.UpdateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external squad", err.Error())
		return
	}
	externalSquadToPlan(updated, &plan)
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

func externalSquadFromPlan(plan *externalSquadModel) (*ExternalSquad, error) {
	squad := &ExternalSquad{Name: plan.Name.ValueString()}
	jsonFields := []struct {
		name  string
		value types.String
		dest  *json.RawMessage
	}{
		{name: "templates", value: plan.Templates, dest: &squad.Templates},
		{name: "subscription_settings", value: plan.SubscriptionSettings, dest: &squad.SubscriptionSettings},
		{name: "host_overrides", value: plan.HostOverrides, dest: &squad.HostOverrides},
		{name: "response_headers", value: plan.ResponseHeaders, dest: &squad.ResponseHeaders},
		{name: "hwid_settings", value: plan.HwidSettings, dest: &squad.HwidSettings},
		{name: "custom_remarks", value: plan.CustomRemarks, dest: &squad.CustomRemarks},
	}
	for _, field := range jsonFields {
		if field.value.IsNull() || field.value.IsUnknown() {
			continue
		}
		value := []byte(field.value.ValueString())
		if !json.Valid(value) {
			return nil, fmt.Errorf("%s must contain valid JSON", field.name)
		}
		*field.dest = json.RawMessage(value)
	}
	if !plan.SubpageConfigUUID.IsNull() && !plan.SubpageConfigUUID.IsUnknown() {
		value := plan.SubpageConfigUUID.ValueString()
		squad.SubpageConfigUUID = &value
	}
	return squad, nil
}

func externalSquadToPlan(squad *ExternalSquad, plan *externalSquadModel) {
	plan.UUID = types.StringValue(squad.UUID)
	plan.Name = types.StringValue(squad.Name)
	plan.Templates = rawJSONToString(squad.Templates)
	plan.SubscriptionSettings = rawJSONToString(squad.SubscriptionSettings)
	plan.HostOverrides = rawJSONToString(squad.HostOverrides)
	plan.ResponseHeaders = rawJSONToString(squad.ResponseHeaders)
	plan.HwidSettings = rawJSONToString(squad.HwidSettings)
	plan.CustomRemarks = rawJSONToString(squad.CustomRemarks)
	if squad.SubpageConfigUUID != nil {
		plan.SubpageConfigUUID = types.StringValue(*squad.SubpageConfigUUID)
	} else {
		plan.SubpageConfigUUID = types.StringNull()
	}
}
