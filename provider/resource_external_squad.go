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
	UUID                  types.String `tfsdk:"uuid"`
	Name                  types.String `tfsdk:"name"`
	Templates             types.String `tfsdk:"templates"`
	SubscriptionSettings  types.String `tfsdk:"subscription_settings"`
	HostOverrides         types.String `tfsdk:"host_overrides"`
	ResponseHeaders       types.String `tfsdk:"response_headers"`
	ResponseHeadersAdd    types.String `tfsdk:"response_headers_add"`
	ResponseHeadersRemove types.String `tfsdk:"response_headers_remove"`
	HwidSettings          types.String `tfsdk:"hwid_settings"`
	CustomRemarks         types.String `tfsdk:"custom_remarks"`
	SubpageConfigUUID     types.String `tfsdk:"subpage_config_uuid"`
}

func NewExternalSquadResource() resource.Resource { return &externalSquadResource{} }

func (r *externalSquadResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_external_squad"
}

func (r *externalSquadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave external squad.",
		Attributes: map[string]schema.Attribute{
			"uuid":                    schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                    schema.StringAttribute{Required: true, Description: "Squad name (2-30 chars)."},
			"templates":               schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Template assignments as JSON array."},
			"subscription_settings":   schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad-specific subscription settings as JSON."},
			"host_overrides":          schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad host overrides as JSON."},
			"response_headers":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalHeaderMapJSONPlanModifier{}}, Description: "Squad response headers as a JSON object on Remnawave 2.x. Header names are treated case-insensitively. Retained as a no-op in state on 3.0+; use response_headers_add and response_headers_remove instead."},
			"response_headers_add":    schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalHeaderMapJSONPlanModifier{}}, Description: "Headers to add to subscription responses, as a JSON object on Remnawave 3.0+. Header names are treated case-insensitively. Retained as a no-op in state on 2.x."},
			"response_headers_remove": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalHeaderListJSONPlanModifier{}}, Description: "JSON array of header names to remove from subscription responses on Remnawave 3.0+. Header names are treated case-insensitively. Retained as a no-op in state on 2.x."},
			"hwid_settings":           schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad HWID settings as JSON."},
			"custom_remarks":          schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Squad custom remarks as JSON."},
			"subpage_config_uuid":     schema.StringAttribute{Optional: true, Computed: true, Description: "Subscription page config UUID assigned to the squad."},
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
	v3, err := r.client.isVersionAtLeast3_0(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine backend version", err.Error())
		return
	}
	configured := plan
	update, err := externalSquadFromPlan(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	if err := adaptExternalSquadRequest(update, v3); err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	warnUnsupportedExternalSquadFields(ctx, &plan, v3)
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
	if err := preserveUnsupportedExternalSquadFields(&configured, &plan, v3); err != nil {
		resp.Diagnostics.AddError("Failed to preserve version-specific external squad state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *externalSquadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state externalSquadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v3, err := r.client.isVersionAtLeast3_0(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine backend version", err.Error())
		return
	}
	previous := state
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
	if err := preserveUnsupportedExternalSquadFields(&previous, &state, v3); err != nil {
		resp.Diagnostics.AddError("Failed to preserve version-specific external squad state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *externalSquadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan externalSquadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v3, err := r.client.isVersionAtLeast3_0(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine backend version", err.Error())
		return
	}
	configured := plan
	squad, err := externalSquadFromPlan(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	if err := adaptExternalSquadRequest(squad, v3); err != nil {
		resp.Diagnostics.AddError("Invalid external squad configuration", err.Error())
		return
	}
	warnUnsupportedExternalSquadFields(ctx, &plan, v3)
	squad.UUID = plan.UUID.ValueString()
	updated, err := r.client.UpdateExternalSquad(ctx, squad)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external squad", err.Error())
		return
	}
	externalSquadToPlan(updated, &plan)
	if err := preserveUnsupportedExternalSquadFields(&configured, &plan, v3); err != nil {
		resp.Diagnostics.AddError("Failed to preserve version-specific external squad state", err.Error())
		return
	}
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
		{name: "response_headers_add", value: plan.ResponseHeadersAdd, dest: &squad.ResponseHeadersAdd},
		{name: "response_headers_remove", value: plan.ResponseHeadersRemove, dest: &squad.ResponseHeadersRemove},
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
	plan.ResponseHeadersAdd = rawJSONToString(squad.ResponseHeadersAdd)
	plan.ResponseHeadersRemove = rawJSONToString(squad.ResponseHeadersRemove)
	plan.HwidSettings = rawJSONToString(squad.HwidSettings)
	plan.CustomRemarks = rawJSONToString(squad.CustomRemarks)
	if squad.SubpageConfigUUID != nil {
		plan.SubpageConfigUUID = types.StringValue(*squad.SubpageConfigUUID)
	} else {
		plan.SubpageConfigUUID = types.StringNull()
	}
}

var removedExternalSubscriptionSettingKeys = map[string]struct{}{
	"profileTitle":               {},
	"supportLink":                {},
	"profileUpdateInterval":      {},
	"isProfileWebpageUrlEnabled": {},
	"happAnnounce":               {},
	"happRouting":                {},
}

func adaptExternalSquadRequest(squad *ExternalSquad, v3 bool) error {
	if !v3 {
		squad.ResponseHeadersAdd = nil
		squad.ResponseHeadersRemove = nil
		return nil
	}

	squad.ResponseHeaders = nil
	filtered, err := filterRemovedExternalSubscriptionSettings(squad.SubscriptionSettings)
	if err != nil {
		return err
	}
	squad.SubscriptionSettings = filtered
	return nil
}

func filterRemovedExternalSubscriptionSettings(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("subscription_settings must be a JSON object")
	}
	for key := range removedExternalSubscriptionSettingKeys {
		delete(object, key)
	}
	filtered, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("marshal subscription_settings: %w", err)
	}
	return filtered, nil
}

func preserveUnsupportedExternalSquadFields(previous, current *externalSquadModel, v3 bool) error {
	if v3 {
		if !previous.ResponseHeaders.IsUnknown() {
			current.ResponseHeaders = previous.ResponseHeaders
		}
		var err error
		current.ResponseHeadersAdd, err = preserveEquivalentHeaderMap(previous.ResponseHeadersAdd, current.ResponseHeadersAdd)
		if err != nil {
			return err
		}
		current.ResponseHeadersRemove, err = preserveEquivalentHeaderList(previous.ResponseHeadersRemove, current.ResponseHeadersRemove)
		if err != nil {
			return err
		}
		merged, err := mergeRemovedExternalSubscriptionSettings(previous.SubscriptionSettings, current.SubscriptionSettings)
		if err != nil {
			return err
		}
		current.SubscriptionSettings = merged
		return nil
	}

	if !previous.ResponseHeadersAdd.IsUnknown() {
		current.ResponseHeadersAdd = previous.ResponseHeadersAdd
	}
	if !previous.ResponseHeadersRemove.IsUnknown() {
		current.ResponseHeadersRemove = previous.ResponseHeadersRemove
	}
	var err error
	current.ResponseHeaders, err = preserveEquivalentHeaderMap(previous.ResponseHeaders, current.ResponseHeaders)
	if err != nil {
		return err
	}
	return nil
}

func mergeRemovedExternalSubscriptionSettings(previous, current types.String) (types.String, error) {
	if previous.IsNull() || previous.IsUnknown() {
		return current, nil
	}
	var previousObject map[string]json.RawMessage
	if err := json.Unmarshal([]byte(previous.ValueString()), &previousObject); err != nil || previousObject == nil {
		return types.StringNull(), fmt.Errorf("subscription_settings in prior state must be a JSON object")
	}

	removed := make(map[string]json.RawMessage)
	for key := range removedExternalSubscriptionSettingKeys {
		if value, ok := previousObject[key]; ok {
			removed[key] = value
		}
	}
	if len(removed) == 0 {
		return current, nil
	}

	currentObject := make(map[string]json.RawMessage)
	if !current.IsNull() && !current.IsUnknown() {
		if err := json.Unmarshal([]byte(current.ValueString()), &currentObject); err != nil || currentObject == nil {
			return types.StringNull(), fmt.Errorf("subscription_settings response must be a JSON object")
		}
	}
	for key, value := range removed {
		currentObject[key] = value
	}
	merged, err := json.Marshal(currentObject)
	if err != nil {
		return types.StringNull(), fmt.Errorf("marshal preserved subscription_settings: %w", err)
	}
	return types.StringValue(string(merged)), nil
}

func warnUnsupportedExternalSquadFields(ctx context.Context, plan *externalSquadModel, v3 bool) {
	if v3 {
		if !plan.ResponseHeaders.IsNull() && !plan.ResponseHeaders.IsUnknown() {
			tflog.Warn(ctx, "external_squad response_headers has no effect on Remnawave 3.0+; use response_headers_add and response_headers_remove")
		}
		if hasRemovedExternalSubscriptionSettings(plan.SubscriptionSettings) {
			tflog.Warn(ctx, "external_squad subscription_settings contains fields removed in Remnawave 3.0+; their values are retained only in Terraform state")
		}
		return
	}
	if !plan.ResponseHeadersAdd.IsNull() && !plan.ResponseHeadersAdd.IsUnknown() {
		tflog.Warn(ctx, "external_squad response_headers_add has no effect on Remnawave 2.x; use response_headers")
	}
	if !plan.ResponseHeadersRemove.IsNull() && !plan.ResponseHeadersRemove.IsUnknown() {
		tflog.Warn(ctx, "external_squad response_headers_remove has no effect on Remnawave 2.x")
	}
}

func hasRemovedExternalSubscriptionSettings(value types.String) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value.ValueString()), &object); err != nil {
		return false
	}
	for key := range removedExternalSubscriptionSettingKeys {
		if _, ok := object[key]; ok {
			return true
		}
	}
	return false
}
