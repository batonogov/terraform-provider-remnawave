package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type hwidDeviceResource struct{ client *Client }

type hwidDeviceModel struct {
	UserUUID types.String `tfsdk:"user_uuid"`
	Hwid     types.String `tfsdk:"hwid"`
	// Optional fields from the backend contract
	Platform    types.String `tfsdk:"platform"`
	OsVersion   types.String `tfsdk:"os_version"`
	DeviceModel types.String `tfsdk:"device_model"`
	UserAgent   types.String `tfsdk:"user_agent"`
	RequestIp   types.String `tfsdk:"request_ip"`
	// Computed
	Id types.String `tfsdk:"id"`
}

func NewHwidDeviceResource() resource.Resource { return &hwidDeviceResource{} }

func (r *hwidDeviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_hwid_device"
}

func (r *hwidDeviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave HWID device entry for a user.",
		Attributes: map[string]schema.Attribute{
			"user_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the user this device belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hwid": schema.StringAttribute{
				Required:    true,
				Description: "Hardware identifier string for the device.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"platform": schema.StringAttribute{
				Optional:    true,
				Description: "Device platform (e.g. android, ios, windows).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_version": schema.StringAttribute{
				Optional:    true,
				Description: "Operating system version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"device_model": schema.StringAttribute{
				Optional:    true,
				Description: "Device model name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_agent": schema.StringAttribute{
				Optional:    true,
				Description: "User agent string.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"request_ip": schema.StringAttribute{
				Optional:    true,
				Description: "Request IP address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier (user_uuid:hwid).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *hwidDeviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func hwidCreateReq(plan *hwidDeviceModel) map[string]any {
	req := map[string]any{
		"userUuid": plan.UserUUID.ValueString(),
		"hwid":     plan.Hwid.ValueString(),
	}
	if !plan.Platform.IsNull() && !plan.Platform.IsUnknown() {
		req["platform"] = plan.Platform.ValueString()
	}
	if !plan.OsVersion.IsNull() && !plan.OsVersion.IsUnknown() {
		req["osVersion"] = plan.OsVersion.ValueString()
	}
	if !plan.DeviceModel.IsNull() && !plan.DeviceModel.IsUnknown() {
		req["deviceModel"] = plan.DeviceModel.ValueString()
	}
	if !plan.UserAgent.IsNull() && !plan.UserAgent.IsUnknown() {
		req["userAgent"] = plan.UserAgent.ValueString()
	}
	if !plan.RequestIp.IsNull() && !plan.RequestIp.IsUnknown() {
		req["requestIp"] = plan.RequestIp.ValueString()
	}
	return req
}

func (r *hwidDeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hwidDeviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.CreateHwidDevice(ctx, hwidCreateReq(&plan)); err != nil {
		resp.Diagnostics.AddError("Failed to create HWID device", err.Error())
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%s:%s", plan.UserUUID.ValueString(), plan.Hwid.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hwidDeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hwidDeviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.client.GetUserHwidDevices(ctx, state.UserUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read HWID devices", err.Error())
		return
	}

	// Check if our device still exists in the list
	devices, ok := data["devices"].([]any)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	found := false
	for _, d := range devices {
		dev, ok := d.(map[string]any)
		if !ok {
			continue
		}
		if hwid, ok := dev["hwid"].(string); ok && hwid == state.Hwid.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hwidDeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace, so Update is never invoked.
	var plan hwidDeviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hwidDeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hwidDeviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := map[string]any{
		"userUuid": state.UserUUID.ValueString(),
		"hwid":     state.Hwid.ValueString(),
	}

	if err := r.client.DeleteHwidDevice(ctx, deleteReq); err != nil {
		resp.Diagnostics.AddError("Failed to delete HWID device", err.Error())
		return
	}
}

func (r *hwidDeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	userUUID, hwid, ok := strings.Cut(req.ID, ":")
	if !ok || userUUID == "" || hwid == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected import ID in user_uuid:hwid format.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_uuid"), types.StringValue(userUUID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hwid"), types.StringValue(hwid))...)
}
