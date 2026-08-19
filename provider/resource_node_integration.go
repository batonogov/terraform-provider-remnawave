package provider

import (
	"context"
	"encoding/json"
	"fmt"

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

type nodeIntegrationResource struct{ client *Client }

type nodeIntegrationResourceModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Config               types.String `tfsdk:"config"`
	RestartNodesOnUpdate types.Bool   `tfsdk:"restart_nodes_on_update"`
}

func NewNodeIntegrationResource() resource.Resource { return &nodeIntegrationResource{} }

func (r *nodeIntegrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node_integration"
}

func (r *nodeIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave 3.3+ node integration.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "UUID of the node integration.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Integration name (2-30 chars).",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 30),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Optional integration description (max 255 chars).",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"config": schema.StringAttribute{
				Required:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}},
				Description:   "Integration configuration as a JSON object. The structure is consumed by Remnawave and the node.",
			},
			"restart_nodes_on_update": schema.BoolAttribute{
				Optional: true,
				Description: "Force-restart affected nodes after updating the integration. Defaults to false; " +
					"the preference is retained in Terraform state but is not persisted by Remnawave.",
			},
		},
	}
}

func (r *nodeIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodeIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if err := r.requireSupport(ctx); err != nil {
		resp.Diagnostics.AddError("Unsupported node integration", err.Error())
		return
	}
	var plan nodeIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration, err := nodeIntegrationFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid node integration", err.Error())
		return
	}
	created, err := r.client.CreateNodeIntegration(ctx, integration)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create node integration", err.Error())
		return
	}
	if err := nodeIntegrationToModel(created, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to decode node integration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if err := r.requireSupport(ctx); err != nil {
		resp.Diagnostics.AddError("Unsupported node integration", err.Error())
		return
	}
	var state nodeIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration, err := r.client.GetNodeIntegrationByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "node integration not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read node integration", err.Error())
		return
	}
	if err := nodeIntegrationToModel(integration, &state); err != nil {
		resp.Diagnostics.AddError("Failed to decode node integration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nodeIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if err := r.requireSupport(ctx); err != nil {
		resp.Diagnostics.AddError("Unsupported node integration", err.Error())
		return
	}
	var plan nodeIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration, err := nodeIntegrationFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid node integration", err.Error())
		return
	}
	integration.UUID = plan.UUID.ValueString()
	if !plan.RestartNodesOnUpdate.IsNull() && !plan.RestartNodesOnUpdate.IsUnknown() {
		integration.RestartNodes = plan.RestartNodesOnUpdate.ValueBool()
	}
	updated, err := r.client.UpdateNodeIntegration(ctx, integration)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update node integration", err.Error())
		return
	}
	if err := nodeIntegrationToModel(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to decode node integration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.requireSupport(ctx); err != nil {
		resp.Diagnostics.AddError("Unsupported node integration", err.Error())
		return
	}
	var state nodeIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNodeIntegration(ctx, state.UUID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete node integration", err.Error())
	}
}

func (r *nodeIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

func (r *nodeIntegrationResource) requireSupport(ctx context.Context) error {
	return requireBackend3_3(ctx, r.client, "node integrations")
}

func requireBackend3_3(ctx context.Context, client *Client, feature string) error {
	supported, err := client.isVersionAtLeast3_3(ctx)
	if err != nil {
		return fmt.Errorf("detect backend version: %w", err)
	}
	if !supported {
		return fmt.Errorf("%s require Remnawave 3.3 or later", feature)
	}
	return nil
}

func nodeIntegrationFromModel(model *nodeIntegrationResourceModel) (*NodeIntegration, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(model.Config.ValueString()), &config); err != nil {
		return nil, fmt.Errorf("config must be a JSON object: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	integration := &NodeIntegration{
		Name:   model.Name.ValueString(),
		Config: config,
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		integration.Description = &description
	}
	return integration, nil
}

func nodeIntegrationToModel(integration *NodeIntegration, model *nodeIntegrationResourceModel) error {
	config, err := json.Marshal(integration.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	model.UUID = types.StringValue(integration.UUID)
	model.Name = types.StringValue(integration.Name)
	if integration.Description != nil {
		model.Description = types.StringValue(*integration.Description)
	} else {
		model.Description = types.StringNull()
	}
	model.Config = types.StringValue(string(config))
	return nil
}
