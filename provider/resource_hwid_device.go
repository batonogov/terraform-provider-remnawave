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
				Description: "Identifier of the user this device belongs to: UUID on Remnawave 2.x, numeric ID on 3.0+.",
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
			// The following metadata attributes are panel-collected when a client
			// connects (device fingerprint, OS, IP, user agent). The operator cannot
			// know them ahead of time and the backend has no Update endpoint, so they
			// are read-only (Computed). Setting them in HCL is rejected; the backend
			// would overwrite them on the next client connection anyway. Only the
			// identity pair (user_uuid + hwid) triggers replacement.
			"platform": schema.StringAttribute{
				Computed:    true,
				Description: "Device platform (e.g. android, ios, windows).",
			},
			"os_version": schema.StringAttribute{
				Computed:    true,
				Description: "Operating system version.",
			},
			"device_model": schema.StringAttribute{
				Computed:    true,
				Description: "Device model name.",
			},
			"user_agent": schema.StringAttribute{
				Computed:    true,
				Description: "User agent string.",
			},
			"request_ip": schema.StringAttribute{
				Computed:    true,
				Description: "Request IP address.",
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

// hwidCreateReq builds the create request. Only the identity pair is sent:
// the metadata attributes are Computed (panel-owned) and never settable in
// config, so they have no source value in the plan. The backend does not offer
// an Update endpoint, so sending them would be pointless — the panel
// overwrites them on the next client connection.
// v3.0+: the request body key is "userId" (number), not "userUuid" (string).
func hwidCreateReq(ctx context.Context, client *Client, plan *hwidDeviceModel) (map[string]any, error) {
	id := plan.UserUUID.ValueString()
	v3, err := client.isVersionAtLeast3_0(ctx)
	if err != nil {
		return nil, err
	}
	if v3 {
		ids, err := stringsToIDs([]string{id})
		if err != nil {
			return nil, fmt.Errorf("user_uuid must be a numeric ID on Remnawave 3.0+, got %q", id)
		}
		return map[string]any{
			"userId": ids[0],
			"hwid":   plan.Hwid.ValueString(),
		}, nil
	}
	return map[string]any{
		"userUuid": id,
		"hwid":     plan.Hwid.ValueString(),
	}, nil
}

func (r *hwidDeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hwidDeviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, err := hwidCreateReq(ctx, r.client, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid user identifier", err.Error())
		return
	}
	if _, err := r.client.CreateHwidDevice(ctx, createReq); err != nil {
		resp.Diagnostics.AddError("Failed to create HWID device", err.Error())
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%s:%s", plan.UserUUID.ValueString(), plan.Hwid.ValueString()))

	// The metadata fields are Computed and have no config source, so they arrive
	// from the plan as unknown markers. Terraform rejects unknown values in the
	// post-apply state ("provider still indicated an unknown value"), so
	// materialize them as null here. The panel populates them asynchronously on
	// the next client connection, at which point Read refreshes them; until then
	// null is the correct value (the device was just registered).
	plan.Platform = types.StringNull()
	plan.OsVersion = types.StringNull()
	plan.DeviceModel = types.StringNull()
	plan.UserAgent = types.StringNull()
	plan.RequestIp = types.StringNull()

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
			if v, ok := dev["platform"].(string); ok && v != "" {
				state.Platform = types.StringValue(v)
			} else {
				state.Platform = types.StringNull()
			}
			if v, ok := dev["osVersion"].(string); ok && v != "" {
				state.OsVersion = types.StringValue(v)
			} else {
				state.OsVersion = types.StringNull()
			}
			if v, ok := dev["deviceModel"].(string); ok && v != "" {
				state.DeviceModel = types.StringValue(v)
			} else {
				state.DeviceModel = types.StringNull()
			}
			if v, ok := dev["userAgent"].(string); ok && v != "" {
				state.UserAgent = types.StringValue(v)
			} else {
				state.UserAgent = types.StringNull()
			}
			if v, ok := dev["requestIp"].(string); ok && v != "" {
				state.RequestIp = types.StringValue(v)
			} else {
				state.RequestIp = types.StringNull()
			}
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
	// All configurable attributes (user_uuid, hwid) use RequiresReplace and the
	// metadata fields are read-only (Computed), so Update is never invoked.
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
		"hwid": state.Hwid.ValueString(),
	}
	v3, err := r.client.isVersionAtLeast3_0(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine backend version", err.Error())
		return
	}
	if v3 {
		ids, err := stringsToIDs([]string{state.UserUUID.ValueString()})
		if err != nil {
			resp.Diagnostics.AddError("Invalid user identifier", fmt.Sprintf("user_uuid must be a numeric ID on Remnawave 3.0+, got %q", state.UserUUID.ValueString()))
			return
		}
		deleteReq["userId"] = ids[0]
	} else {
		deleteReq["userUuid"] = state.UserUUID.ValueString()
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
