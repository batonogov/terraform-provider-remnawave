package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type subscriptionTemplateResource struct{ client *Client }
type subscriptionTemplateModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	TemplateType types.String `tfsdk:"template_type"`
	TemplateJSON types.String `tfsdk:"template_json"`
	EncodedYaml  types.String `tfsdk:"encoded_template_yaml"`
}

func NewSubscriptionTemplateResource() resource.Resource { return &subscriptionTemplateResource{} }

func (r *subscriptionTemplateResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_subscription_template"
}

func (r *subscriptionTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave subscription template.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Required: true, Description: "Template name (2-255 chars)."},
			"template_type": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			}, Description: "Type: XRAY_JSON, XRAY_BASE64, MIHOMO, STASH, CLASH, SINGBOX."},
			"template_json":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Template JSON (opaque)."},
			"encoded_template_yaml": schema.StringAttribute{Optional: true, Computed: true, Description: "Encoded template YAML."},
		},
	}
}

func (r *subscriptionTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subscriptionTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subscriptionTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var update *SubscriptionTemplate
	if (!plan.TemplateJSON.IsNull() && !plan.TemplateJSON.IsUnknown()) || (!plan.EncodedYaml.IsNull() && !plan.EncodedYaml.IsUnknown()) {
		update = &SubscriptionTemplate{Name: plan.Name.ValueString()}
		if !plan.TemplateJSON.IsNull() && !plan.TemplateJSON.IsUnknown() {
			canonical, err := canonicalJSONString(plan.TemplateJSON.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid template_json", err.Error())
				return
			}
			plan.TemplateJSON = types.StringValue(canonical)
			if err := json.Unmarshal([]byte(canonical), &update.TemplateJSON); err != nil {
				resp.Diagnostics.AddError("Invalid template_json", err.Error())
				return
			}
		}
		if !plan.EncodedYaml.IsNull() && !plan.EncodedYaml.IsUnknown() {
			update.EncodedTemplateYaml = plan.EncodedYaml.ValueString()
		}
	}
	tmpl := &SubscriptionTemplate{Name: plan.Name.ValueString(), TemplateType: plan.TemplateType.ValueString()}
	created, err := r.client.CreateSubscriptionTemplate(ctx, tmpl)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create subscription template", err.Error())
		return
	}
	current := created
	if update != nil {
		update.UUID = created.UUID
		current, err = r.client.UpdateSubscriptionTemplate(ctx, update)
		if err != nil {
			detail := err.Error()
			if cleanupErr := r.client.DeleteSubscriptionTemplate(ctx, created.UUID); cleanupErr != nil {
				detail += fmt.Sprintf("; additionally failed to delete partially created template %s: %v", created.UUID, cleanupErr)
			}
			resp.Diagnostics.AddError("Failed to set subscription template content", detail)
			return
		}
	}
	if !subscriptionTemplateToPlan(current, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subscriptionTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tmpl, err := r.client.GetSubscriptionTemplateByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "subscription template not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read subscription template", err.Error())
		return
	}
	if !subscriptionTemplateToPlan(tmpl, &state, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriptionTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan subscriptionTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tmpl := &SubscriptionTemplate{UUID: plan.UUID.ValueString(), Name: plan.Name.ValueString()}
	if !plan.TemplateJSON.IsNull() {
		var cfg any
		if err := json.Unmarshal([]byte(plan.TemplateJSON.ValueString()), &cfg); err != nil {
			resp.Diagnostics.AddError("Invalid template_json", err.Error())
			return
		}
		tmpl.TemplateJSON = cfg
	}
	if !plan.EncodedYaml.IsNull() {
		tmpl.EncodedTemplateYaml = plan.EncodedYaml.ValueString()
	}
	updated, err := r.client.UpdateSubscriptionTemplate(ctx, tmpl)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update subscription template", err.Error())
		return
	}
	if !subscriptionTemplateToPlan(updated, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subscriptionTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSubscriptionTemplate(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete subscription template", err.Error())
	}
}

func (r *subscriptionTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

func subscriptionTemplateToPlan(template *SubscriptionTemplate, plan *subscriptionTemplateModel, diagnostics *diag.Diagnostics) bool {
	plan.UUID = types.StringValue(template.UUID)
	plan.Name = types.StringValue(template.Name)
	plan.TemplateType = types.StringValue(template.TemplateType)
	if template.TemplateJSON != nil {
		b, err := json.Marshal(template.TemplateJSON)
		if err != nil {
			diagnostics.AddError("Failed to marshal template_json", err.Error())
			return false
		}
		plan.TemplateJSON = types.StringValue(string(b))
	} else {
		plan.TemplateJSON = types.StringNull()
	}
	if template.EncodedTemplateYaml != "" {
		plan.EncodedYaml = types.StringValue(template.EncodedTemplateYaml)
	} else {
		plan.EncodedYaml = types.StringNull()
	}
	return true
}
