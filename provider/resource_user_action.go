package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// validUserActions is the set of actions accepted by the resource.
// "reset-traffic" is accepted as a backward-compatible alias for the
// canonical "reset_traffic" form (see normalizeUserAction).
var validUserActions = map[string]struct{}{
	"enable":              {},
	"disable":             {},
	"reset_traffic":       {},
	"reset-traffic":       {}, // alias, deprecated
	"revoke_subscription": {},
}

// userActionAliases maps deprecated/alias action names to their canonical
// (underscore) form. An entry that maps to itself (or is absent) needs no
// rewriting.
var userActionAliases = map[string]string{
	"reset-traffic": "reset_traffic",
}

// normalizeUserAction returns the canonical form of action. If action is a
// known alias (e.g. "reset-traffic"), the canonical name is returned and
// warned is set to true so the caller can emit a deprecation warning.
func normalizeUserAction(action string) (canonical string, warned bool) {
	if c, ok := userActionAliases[action]; ok {
		return c, true
	}
	return action, false
}

type userActionResource struct {
	client *Client
}

type userActionModel struct {
	ID       types.String `tfsdk:"id"`
	UserUUID types.String `tfsdk:"user_uuid"`
	Action   types.String `tfsdk:"action"`
	Triggers types.List   `tfsdk:"triggers"`
}

func NewUserActionResource() resource.Resource {
	return &userActionResource{}
}

func (r *userActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_user_action"
}

func (r *userActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs an imperative one-shot action on a Remnawave user (enable, disable, reset_traffic, or revoke_subscription). " +
			"The action is re-executed whenever the `triggers` list changes value, making it suitable for recurring operations such as periodic traffic resets. " +
			"`reset-traffic` is accepted as a backward-compatible alias for `reset_traffic` (prefer the underscore form).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier: <user_uuid>:<action>.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the target user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: "Action to perform. One of: enable, disable, reset_traffic, revoke_subscription. `reset-traffic` is accepted as a backward-compatible alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Arbitrary list of strings. When any value changes, the action is re-executed. " +
					"Commonly set to `[timestamp()]` for periodic re-runs.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
		},
	}
}

func (r *userActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action := plan.Action.ValueString()
	if !isValidUserAction(action) {
		resp.Diagnostics.AddError(
			"Invalid action",
			fmt.Sprintf("action must be one of: enable, disable, reset_traffic, revoke_subscription — got %q", action),
		)
		return
	}

	// Normalize alias spellings (e.g. "reset-traffic" → "reset_traffic") to
	// the canonical underscore form and warn about deprecated usage.
	canonicalAction, warnAlias := normalizeUserAction(action)
	if warnAlias {
		tflog.Warn(ctx, "user_action uses deprecated hyphenated form; prefer the underscore form (will keep working but may be removed in a future release)", map[string]any{
			"action":           action,
			"canonical_action": canonicalAction,
			"user_uuid":        plan.UserUUID.ValueString(),
		})
	}
	// NOTE: do NOT overwrite plan.Action — Terraform requires state to match
	// the config value the user supplied. The alias is preserved in state
	// and only normalized for the API call.
	action = canonicalAction

	userUUID := plan.UserUUID.ValueString()
	tflog.Info(ctx, "performing user action", map[string]any{
		"user_uuid": userUUID,
		"action":    action,
	})

	if err := r.client.UserAction(ctx, userUUID, action); err != nil {
		resp.Diagnostics.AddError("Failed to perform user action", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", userUUID, action))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userActionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the target user still exists; if not, remove from state.
	userUUID := state.UserUUID.ValueString()
	if userUUID == "" {
		return
	}

	if _, err := r.client.GetUserByUUID(ctx, userUUID); err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "user not found, removing user_action from state", map[string]any{"user_uuid": userUUID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read user for user_action", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// user_uuid and action both use RequiresReplace. The only attribute that
	// can change in-place is triggers, and triggers uses RequiresReplaceIfConfigured,
	// which means Terraform destroys-and-recreates rather than calling Update.
	// This method should never be invoked in practice.
	var plan userActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userActionResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: user actions are imperative one-shot operations with no
	// corresponding "undo" endpoint. Removing from state is sufficient.
}

func (r *userActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	userUUID, action, ok := strings.Cut(req.ID, ":")
	if !ok || userUUID == "" || action == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected import ID in <user_uuid>:<action> format.")
		return
	}
	if !isValidUserAction(action) {
		resp.Diagnostics.AddError(
			"Invalid action in import ID",
			fmt.Sprintf("action must be one of: enable, disable, reset_traffic, revoke_subscription — got %q", action),
		)
		return
	}
	canonicalAction, warnAlias := normalizeUserAction(action)
	if warnAlias {
		tflog.Warn(ctx, "user_action import uses deprecated hyphenated form; prefer the underscore form (will keep working but may be removed in a future release)", map[string]any{
			"action":           action,
			"canonical_action": canonicalAction,
			"user_uuid":        userUUID,
		})
		action = canonicalAction
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(fmt.Sprintf("%s:%s", userUUID, action)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_uuid"), types.StringValue(userUUID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("action"), types.StringValue(action))...)
}

// isValidUserAction returns true if action is one of the supported user actions.
func isValidUserAction(action string) bool {
	_, ok := validUserActions[action]
	return ok
}
