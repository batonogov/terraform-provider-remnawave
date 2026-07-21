package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Node Bulk Action Resource ───

// validNodeBulkActions defines the accepted action values.
var validNodeBulkActions = map[string]bool{
	"enable":        true,
	"disable":       true,
	"restart":       true,
	"reset_traffic": true,
}

type nodeBulkActionResource struct {
	client *Client
}

type nodeBulkActionResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Action   types.String `tfsdk:"action"`
	UUIDs    types.List   `tfsdk:"uuids"`
	Triggers types.Map    `tfsdk:"triggers"`
}

func NewNodeBulkActionResource() resource.Resource {
	return &nodeBulkActionResource{}
}

func (r *nodeBulkActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node_bulk_action"
}

func (r *nodeBulkActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs a bulk operation (enable, disable, restart, or reset_traffic) on " +
			"one or more nodes. This is an imperative resource: the action runs when the resource " +
			"is created and whenever its arguments change; an apply with no changes does not " +
			"repeat it. Change `triggers` to force re-execution without changing the operation inputs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Opaque identifier for this bulk action resource instance.",
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: "Bulk action to perform. One of: `enable`, `disable`, `restart`, `reset_traffic`.",
			},
			"uuids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of node UUIDs to operate on.",
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A map of arbitrary key/value pairs. When any value changes, " +
					"the resource is re-applied and the action re-executed.",
			},
		},
	}
}

func (r *nodeBulkActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodeBulkActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nodeBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action := plan.Action.ValueString()
	if !validNodeBulkActions[action] {
		resp.Diagnostics.AddError(
			"Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, restart, reset_traffic — got %q", action),
		)
		return
	}

	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute node bulk action", err.Error())
		return
	}

	plan.ID = types.StringValue(nodeBulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeBulkActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Imperative resource: nothing to read back from the API.
	var state nodeBulkActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nodeBulkActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nodeBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action := plan.Action.ValueString()
	if !validNodeBulkActions[action] {
		resp.Diagnostics.AddError(
			"Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, restart, reset_traffic — got %q", action),
		)
		return
	}

	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute node bulk action", err.Error())
		return
	}

	plan.ID = types.StringValue(nodeBulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeBulkActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: this is an imperative resource. We do not reverse the action on destroy.
}

// execute calls the node bulk endpoint with the given action string.
func (r *nodeBulkActionResource) execute(ctx context.Context, plan *nodeBulkActionResourceModel) error {
	uuids := make([]string, 0, len(plan.UUIDs.Elements()))
	for _, el := range plan.UUIDs.Elements() {
		uuids = append(uuids, el.(types.String).ValueString())
	}
	return r.client.BulkNodeAction(ctx, plan.Action.ValueString(), uuids)
}

// nodeBulkActionID produces a deterministic ID for the resource instance.
func nodeBulkActionID(plan *nodeBulkActionResourceModel) string {
	parts := make([]string, 0, len(plan.UUIDs.Elements())+1)
	parts = append(parts, plan.Action.ValueString())
	for _, el := range plan.UUIDs.Elements() {
		parts = append(parts, el.(types.String).ValueString())
	}
	return strings.Join(parts, ":")
}
