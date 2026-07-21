package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── User Bulk Action Resource ───

// validUserBulkActions defines the accepted action values.
var validUserBulkActions = map[string]bool{
	"reset_traffic":       true,
	"revoke_subscription": true,
	"delete":              true,
	"extend_expiration":   true,
}

type userBulkActionResource struct {
	client *Client
}

type userBulkActionResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Action   types.String `tfsdk:"action"`
	UUIDs    types.List   `tfsdk:"uuids"`
	Days     types.Int64  `tfsdk:"days"`
	Triggers types.Map    `tfsdk:"triggers"`
}

func NewUserBulkActionResource() resource.Resource {
	return &userBulkActionResource{}
}

func (r *userBulkActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_user_bulk_action"
}

func (r *userBulkActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs a bulk operation (reset_traffic, revoke_subscription, delete, or " +
			"extend_expiration) on one or more users. This is an imperative resource: the action " +
			"runs when the resource is created and whenever its arguments change; an apply with no " +
			"changes does not repeat it. Change `triggers` to force re-execution without changing " +
			"the operation inputs.",
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
				Description: "Bulk action to perform. One of: `reset_traffic`, `revoke_subscription`, `delete`, `extend_expiration`.",
			},
			"uuids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of user UUIDs to operate on.",
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 500),
				},
			},
			"days": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of days (1-9999) to extend expiration. Required when action is `extend_expiration`; ignored otherwise.",
				Validators: []validator.Int64{
					int64validator.Between(1, 9999),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
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

func (r *userBulkActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userBulkActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validate(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid user bulk action configuration", err.Error())
		return
	}

	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute user bulk action", err.Error())
		return
	}

	plan.ID = types.StringValue(userBulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userBulkActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Imperative resource: nothing to read back from the API.
	var state userBulkActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userBulkActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userBulkActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validate(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid user bulk action configuration", err.Error())
		return
	}

	if err := r.execute(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to execute user bulk action", err.Error())
		return
	}

	plan.ID = types.StringValue(userBulkActionID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userBulkActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: this is an imperative resource. We do not reverse the action on destroy.
}

// validate ensures the action is known and that days is provided when the
// action is extend_expiration.
func (r *userBulkActionResource) validate(plan *userBulkActionResourceModel) error {
	action := plan.Action.ValueString()
	if !validUserBulkActions[action] {
		return fmt.Errorf("action must be one of: reset_traffic, revoke_subscription, delete, extend_expiration — got %q", action)
	}
	count := len(plan.UUIDs.Elements())
	if plan.UUIDs.IsNull() || plan.UUIDs.IsUnknown() || count < 1 || count > 500 {
		return fmt.Errorf("uuids must contain between 1 and 500 values")
	}
	if action == "extend_expiration" {
		if plan.Days.IsNull() || plan.Days.IsUnknown() {
			return fmt.Errorf("days is required when action is %q", action)
		}
		days := plan.Days.ValueInt64()
		if days < 1 || days > 9999 {
			return fmt.Errorf("days must be between 1 and 9999")
		}
	}
	return nil
}

// execute calls the appropriate user bulk endpoint based on the action string.
func (r *userBulkActionResource) execute(ctx context.Context, plan *userBulkActionResourceModel) error {
	uuids := make([]string, 0, len(plan.UUIDs.Elements()))
	for _, el := range plan.UUIDs.Elements() {
		uuids = append(uuids, el.(types.String).ValueString())
	}

	if plan.Action.ValueString() == "extend_expiration" {
		return r.client.BulkUserExtendExpiration(ctx, uuids, int(plan.Days.ValueInt64()))
	}
	return r.client.BulkUserAction(ctx, plan.Action.ValueString(), uuids)
}

// userBulkActionID produces a deterministic ID for the resource instance.
func userBulkActionID(plan *userBulkActionResourceModel) string {
	parts := make([]string, 0, len(plan.UUIDs.Elements())+2)
	parts = append(parts, plan.Action.ValueString())
	if plan.Action.ValueString() == "extend_expiration" && !plan.Days.IsNull() {
		parts = append(parts, fmt.Sprintf("days=%d", plan.Days.ValueInt64()))
	}
	for _, el := range plan.UUIDs.Elements() {
		parts = append(parts, el.(types.String).ValueString())
	}
	return strings.Join(parts, ":")
}
