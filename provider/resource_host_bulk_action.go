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

// ─── Host Bulk Action Resource ───

// validBulkActions defines the accepted action values.
var validBulkActions = map[string]bool{
	"enable":  true,
	"disable": true,
	"delete":  true,
}

type hostBulkActionResource struct {
	client *Client
}

type hostBulkActionResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Action   types.String `tfsdk:"action"`
	UUIDs    types.List   `tfsdk:"uuids"`
	Triggers types.Map    `tfsdk:"triggers"`
}

func NewHostBulkActionResource() resource.Resource {
	return &hostBulkActionResource{}
}

func (r *hostBulkActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_host_bulk_action"
}

func (r *hostBulkActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs a bulk operation (enable, disable, or delete) on one or more hosts. " +
			"This is an imperative resource: the action runs when the resource is created and " +
			"whenever its arguments change; an apply with no changes does not repeat it. " +
			"Change `triggers` to force re-execution without changing the operation inputs.",
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
				Description: "Bulk action to perform. One of: `enable`, `disable`, `delete`.",
			},
			"uuids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of host UUIDs to operate on.",
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

func (r *hostBulkActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostBulkActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action := plan.Action.ValueString()
	if !validBulkActions[action] {
		resp.Diagnostics.AddError(
			"Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, delete — got %q", action),
		)
		return
	}

	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute host bulk action", err.Error())
		return
	}

	// Use a deterministic ID so the resource is addressable.
	plan.ID = types.StringValue(bulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostBulkActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Imperative resource: nothing to read back from the API.
	var state hostBulkActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostBulkActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action := plan.Action.ValueString()
	if !validBulkActions[action] {
		resp.Diagnostics.AddError(
			"Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, delete — got %q", action),
		)
		return
	}

	// Any change to action, uuids, or triggers causes re-execution.
	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute host bulk action", err.Error())
		return
	}

	plan.ID = types.StringValue(bulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostBulkActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: this is an imperative resource. We do not reverse the action on destroy.
}

// execute calls the appropriate bulk endpoint based on the action string.
func (r *hostBulkActionResource) execute(ctx context.Context, plan *hostBulkActionResourceModel) error {
	uuids := make([]string, 0, len(plan.UUIDs.Elements()))
	for _, el := range plan.UUIDs.Elements() {
		uuids = append(uuids, el.(types.String).ValueString())
	}

	switch plan.Action.ValueString() {
	case "enable":
		return r.client.BulkEnableHosts(ctx, uuids)
	case "disable":
		return r.client.BulkDisableHosts(ctx, uuids)
	case "delete":
		return r.client.BulkDeleteHosts(ctx, uuids)
	default:
		return fmt.Errorf("unknown action %q", plan.Action.ValueString())
	}
}

// bulkActionID produces a deterministic ID for the resource instance.
func bulkActionID(plan *hostBulkActionResourceModel) string {
	parts := make([]string, 0, len(plan.UUIDs.Elements())+1)
	parts = append(parts, plan.Action.ValueString())
	for _, el := range plan.UUIDs.Elements() {
		parts = append(parts, el.(types.String).ValueString())
	}
	return strings.Join(parts, ":")
}
