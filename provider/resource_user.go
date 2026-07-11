package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type userResource struct {
	client *Client
}

type userResourceModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	ID                   types.Int64  `tfsdk:"id"`
	ShortUUID            types.String `tfsdk:"short_uuid"`
	Username             types.String `tfsdk:"username"`
	Status               types.String `tfsdk:"status"`
	TrafficLimitBytes    types.Int64  `tfsdk:"traffic_limit_bytes"`
	TrafficLimitStrategy types.String `tfsdk:"traffic_limit_strategy"`
	ExpireAt             types.String `tfsdk:"expire_at"`
	TrojanPassword       types.String `tfsdk:"trojan_password"`
	VlessUUID            types.String `tfsdk:"vless_uuid"`
	SsPassword           types.String `tfsdk:"ss_password"`
	Description          types.String `tfsdk:"description"`
	Tag                  types.String `tfsdk:"tag"`
	TelegramID           types.Int64  `tfsdk:"telegram_id"`
	Email                types.String `tfsdk:"email"`
	HwidDeviceLimit      types.Int64  `tfsdk:"hwid_device_limit"`
	SubscriptionURL      types.String `tfsdk:"subscription_url"`
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave VPN user.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the user (assigned by the panel).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Numeric ID of the user.",
			},
			"short_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "Short UUID used in subscription URLs.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "Unique username (3-36 chars, alphanumeric + _ -).",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "User status: ACTIVE or DISABLED. LIMITED/EXPIRED are managed by the panel.",
			},
			"traffic_limit_bytes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Traffic limit in bytes. 0 = unlimited.",
			},
			"traffic_limit_strategy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Reset strategy: NO_RESET, DAY, WEEK, MONTH, MONTH_ROLLING.",
			},
			"expire_at": schema.StringAttribute{
				Required:    true,
				Description: "Expiration date in ISO 8601 format.",
			},
			"trojan_password": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: "Trojan protocol password (8-32 chars). Auto-generated if not set.",
			},
			"vless_uuid": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "VLESS UUID. Auto-generated if not set.",
			},
			"ss_password": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: "Shadowsocks password (8-32 chars). Auto-generated if not set.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form description.",
			},
			"tag": schema.StringAttribute{
				Optional:    true,
				Description: "User tag (uppercase letters, numbers, underscores; max 16).",
			},
			"telegram_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Telegram user ID for notifications.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "User email address.",
			},
			"hwid_device_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Max hardware devices allowed.",
			},
			"subscription_url": schema.StringAttribute{
				Computed:    true,
				Description: "Subscription URL for the user.",
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := planToUser(&plan)
	created, err := r.client.CreateUser(ctx, user)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	userToPlan(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		return
	}

	user, err := r.client.GetUserByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "user not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}

	userToPlan(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := planToUser(&plan)
	user.UUID = plan.UUID.ValueString()

	updated, err := r.client.UpdateUser(ctx, user)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user", err.Error())
		return
	}

	userToPlan(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if err := r.client.DeleteUser(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

// ─── Conversions ───

func planToUser(p *userResourceModel) *User {
	u := &User{
		Username:             p.Username.ValueString(),
		Status:               p.Status.ValueString(),
		TrafficLimitBytes:    p.TrafficLimitBytes.ValueInt64(),
		TrafficLimitStrategy: p.TrafficLimitStrategy.ValueString(),
		ExpireAt:             p.ExpireAt.ValueString(),
		TrojanPassword:       p.TrojanPassword.ValueString(),
		VlessUUID:            p.VlessUUID.ValueString(),
		SsPassword:           p.SsPassword.ValueString(),
	}
	if !p.Description.IsNull() {
		d := p.Description.ValueString()
		u.Description = &d
	}
	if !p.Tag.IsNull() {
		t := p.Tag.ValueString()
		u.Tag = &t
	}
	if !p.TelegramID.IsNull() {
		t := p.TelegramID.ValueInt64()
		u.TelegramID = &t
	}
	if !p.Email.IsNull() {
		e := p.Email.ValueString()
		u.Email = &e
	}
	if !p.HwidDeviceLimit.IsNull() {
		h := p.HwidDeviceLimit.ValueInt64()
		u.HwidDeviceLimit = &h
	}
	return u
}

func userToPlan(u *User, p *userResourceModel) {
	p.UUID = types.StringValue(u.UUID)
	p.ID = types.Int64Value(u.ID)
	p.ShortUUID = types.StringValue(u.ShortUUID)
	p.Username = types.StringValue(u.Username)
	p.ExpireAt = types.StringValue(u.ExpireAt)
	p.SubscriptionURL = types.StringValue(u.SubscriptionURL)

	if u.Status != "" {
		p.Status = types.StringValue(u.Status)
	}
	if u.TrafficLimitStrategy != "" {
		p.TrafficLimitStrategy = types.StringValue(u.TrafficLimitStrategy)
	}
	if u.TrojanPassword != "" {
		p.TrojanPassword = types.StringValue(u.TrojanPassword)
	}
	if u.VlessUUID != "" {
		p.VlessUUID = types.StringValue(u.VlessUUID)
	}
	if u.SsPassword != "" {
		p.SsPassword = types.StringValue(u.SsPassword)
	}

	p.TrafficLimitBytes = types.Int64Value(u.TrafficLimitBytes)

	if u.Description != nil {
		p.Description = types.StringValue(*u.Description)
	} else {
		p.Description = types.StringNull()
	}
	if u.Tag != nil {
		p.Tag = types.StringValue(*u.Tag)
	} else {
		p.Tag = types.StringNull()
	}
	if u.TelegramID != nil {
		p.TelegramID = types.Int64Value(*u.TelegramID)
	} else {
		p.TelegramID = types.Int64Null()
	}
	if u.Email != nil {
		p.Email = types.StringValue(*u.Email)
	} else {
		p.Email = types.StringNull()
	}
	if u.HwidDeviceLimit != nil {
		p.HwidDeviceLimit = types.Int64Value(*u.HwidDeviceLimit)
	} else {
		p.HwidDeviceLimit = types.Int64Null()
	}
}

// isNotFound checks if the error is a 404 or record-not-found response.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "status 404")
}
