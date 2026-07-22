package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type nodeResource struct {
	client *Client
}

type nodeResourceModel struct {
	UUID                      types.String  `tfsdk:"uuid"`
	Name                      types.String  `tfsdk:"name"`
	Address                   types.String  `tfsdk:"address"`
	Port                      types.Int64   `tfsdk:"port"`
	ProxyURL                  types.String  `tfsdk:"proxy_url"`
	IsTrafficTrackingActive   types.Bool    `tfsdk:"is_traffic_tracking_active"`
	TrafficLimitBytes         types.Int64   `tfsdk:"traffic_limit_bytes"`
	TrafficUsedBytes          types.Int64   `tfsdk:"traffic_used_bytes"`
	TrafficResetDay           types.Int64   `tfsdk:"traffic_reset_day"`
	NotifyPercent             types.Int64   `tfsdk:"notify_percent"`
	CountryCode               types.String  `tfsdk:"country_code"`
	ConsumptionMultiplier     types.Float64 `tfsdk:"consumption_multiplier"`
	NodeConsumptionMultiplier types.Float64 `tfsdk:"node_consumption_multiplier"`
	Tags                      types.Set     `tfsdk:"tags"`
	ProviderUUID              types.String  `tfsdk:"provider_uuid"`
	ActivePluginUUID          types.String  `tfsdk:"active_plugin_uuid"`
	IsConnected               types.Bool    `tfsdk:"is_connected"`
	IsDisabled                types.Bool    `tfsdk:"is_disabled"`
	IsConnecting              types.Bool    `tfsdk:"is_connecting"`
	LastStatusChange          types.String  `tfsdk:"last_status_change"`
	LastStatusMessage         types.String  `tfsdk:"last_status_message"`
	UsersOnline               types.Int64   `tfsdk:"users_online"`
	ViewPosition              types.Int64   `tfsdk:"view_position"`
	XrayUptime                types.Float64 `tfsdk:"xray_uptime"`
	ProviderDetails           types.String  `tfsdk:"provider_details"`
	System                    types.String  `tfsdk:"system"`
	Versions                  types.String  `tfsdk:"versions"`
	CreatedAt                 types.String  `tfsdk:"created_at"`
	UpdatedAt                 types.String  `tfsdk:"updated_at"`
	Note                      types.String  `tfsdk:"note"`
	// config_profile_uuid + config_profile_inbounds are required for create
	ConfigProfileUUID     types.String `tfsdk:"config_profile_uuid"`
	ConfigProfileInbounds types.Set    `tfsdk:"config_profile_inbounds"`
}

func NewNodeResource() resource.Resource {
	return &nodeResource{}
}

func (r *nodeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node"
}

func (r *nodeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave node (Xray server).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "UUID of the node (assigned by the panel).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Node name (3-30 chars).",
			},
			"address": schema.StringAttribute{
				Required:    true,
				Description: "Node address (IP or hostname).",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "Node port for internal panel API communication.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Optional SOCKS5 proxy URL used to reach the node.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^socks5://`),
						"proxy_url must start with socks5://",
					),
				},
			},
			"is_traffic_tracking_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable traffic tracking for this node.",
			},
			"traffic_limit_bytes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Traffic limit in bytes for this node.",
			},
			"traffic_used_bytes": schema.Int64Attribute{
				Computed:    true,
				Description: "Traffic consumed by the node according to the panel.",
			},
			"traffic_reset_day": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Day of month (1-31) to reset traffic counter.",
				Validators: []validator.Int64{
					int64validator.Between(1, 31),
				},
			},
			"notify_percent": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Notify at this traffic usage percentage (0-100).",
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"country_code": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ISO 3166-1 alpha-2 country code (2 chars).",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Z]{2}$`),
						"country_code must be a 2-letter uppercase ISO 3166-1 alpha-2 code",
					),
				},
			},
			"consumption_multiplier": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "User traffic consumption multiplier (0.0-100.0).",
			},
			"node_consumption_multiplier": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Node traffic consumption multiplier (0.0-100.0).",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Node tags (up to 10).",
			},
			"provider_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "Infrastructure provider UUID associated with the node.",
			},
			"active_plugin_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "Active node plugin UUID.",
			},
			"is_connected": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the node is currently connected.",
			},
			"is_disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the node is administratively disabled.",
			},
			"is_connecting": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the node is in the process of connecting.",
			},
			"last_status_change": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the most recent connection status change.",
			},
			"last_status_message": schema.StringAttribute{
				Computed:    true,
				Description: "Most recent connection status message.",
			},
			"users_online": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of users currently online on this node.",
			},
			"view_position": schema.Int64Attribute{Computed: true, Description: "Panel ordering position."},
			"xray_uptime":   schema.Float64Attribute{Computed: true, Description: "Xray uptime reported by the node."},
			"provider_details": schema.StringAttribute{
				Computed:    true,
				Description: "Associated infrastructure provider summary as JSON.",
			},
			"system": schema.StringAttribute{
				Computed:    true,
				Description: "Node system information and live statistics as JSON.",
			},
			"versions": schema.StringAttribute{
				Computed:    true,
				Description: "Xray and node component versions as JSON.",
			},
			"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
			"updated_at": schema.StringAttribute{Computed: true, Description: "Last update timestamp."},
			"note": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form note (max 255 chars).",
			},
			"config_profile_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the config profile assigned to this node.",
			},
			"config_profile_inbounds": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Set of inbound UUIDs enabled for this node's config profile. When omitted, the prior state value is preserved, preventing accidental removal of all active inbounds on update.",
			},
		},
	}
}

func (r *nodeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	node := planToNode(&plan)
	created, err := r.client.CreateNode(ctx, node)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create node", err.Error())
		return
	}

	nodeToPlan(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		return
	}

	node, err := r.client.GetNodeByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "node not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read node", err.Error())
		return
	}

	nodeToPlan(node, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	node := planToNode(&plan)
	node.UUID = plan.UUID.ValueString()

	updated, err := r.client.UpdateNode(ctx, node)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update node", err.Error())
		return
	}

	nodeToPlan(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if err := r.client.DeleteNode(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Failed to delete node", err.Error())
		return
	}
}

func (r *nodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

// ─── Conversions ───

func planToNode(p *nodeResourceModel) *Node {
	n := &Node{
		UUID:                    p.UUID.ValueString(),
		Name:                    p.Name.ValueString(),
		Address:                 p.Address.ValueString(),
		IsTrafficTrackingActive: p.IsTrafficTrackingActive.ValueBool(),
		CountryCode:             p.CountryCode.ValueString(),
	}
	if !p.Port.IsNull() {
		port := int(p.Port.ValueInt64())
		n.Port = &port
	}
	if !p.ProxyURL.IsNull() && !p.ProxyURL.IsUnknown() {
		proxyURL := p.ProxyURL.ValueString()
		n.ProxyURL = &proxyURL
	}
	if !p.TrafficLimitBytes.IsNull() && !p.TrafficLimitBytes.IsUnknown() {
		tl := p.TrafficLimitBytes.ValueInt64()
		n.TrafficLimitBytes = &tl
	}
	if !p.TrafficResetDay.IsNull() && !p.TrafficResetDay.IsUnknown() {
		rd := int(p.TrafficResetDay.ValueInt64())
		n.TrafficResetDay = &rd
	}
	if !p.NotifyPercent.IsNull() && !p.NotifyPercent.IsUnknown() {
		np := int(p.NotifyPercent.ValueInt64())
		n.NotifyPercent = &np
	}
	if !p.ConsumptionMultiplier.IsNull() && !p.ConsumptionMultiplier.IsUnknown() {
		value := p.ConsumptionMultiplier.ValueFloat64()
		n.ConsumptionMultiplier = &value
	}
	if !p.NodeConsumptionMultiplier.IsNull() && !p.NodeConsumptionMultiplier.IsUnknown() {
		value := p.NodeConsumptionMultiplier.ValueFloat64()
		n.NodeConsumptionMultiplier = &value
	}
	if !p.Tags.IsNull() && !p.Tags.IsUnknown() {
		for _, value := range p.Tags.Elements() {
			n.Tags = append(n.Tags, value.(types.String).ValueString())
		}
	}
	if !p.ProviderUUID.IsNull() && !p.ProviderUUID.IsUnknown() {
		value := p.ProviderUUID.ValueString()
		n.ProviderUUID = &value
	}
	if !p.ActivePluginUUID.IsNull() && !p.ActivePluginUUID.IsUnknown() {
		value := p.ActivePluginUUID.ValueString()
		n.ActivePluginUUID = &value
	}
	if !p.Note.IsNull() {
		note := p.Note.ValueString()
		n.Note = &note
	}
	if !p.ConfigProfileUUID.IsNull() && p.ConfigProfileUUID.ValueString() != "" {
		inbounds := []NodeConfigProfileInbound{}
		if !p.ConfigProfileInbounds.IsNull() {
			for _, v := range p.ConfigProfileInbounds.Elements() {
				inbounds = append(inbounds, NodeConfigProfileInbound{UUID: v.(types.String).ValueString()})
			}
		}
		n.ConfigProfile = &NodeConfigProfile{
			ActiveConfigProfileUUID: p.ConfigProfileUUID.ValueString(),
			ActiveInbounds:          inbounds,
		}
	}
	return n
}

func nodeToPlan(n *Node, p *nodeResourceModel) {
	p.UUID = types.StringValue(n.UUID)
	p.Name = types.StringValue(n.Name)
	p.Address = types.StringValue(n.Address)
	p.IsConnected = types.BoolValue(n.IsConnected)
	p.IsDisabled = types.BoolValue(n.IsDisabled)
	p.IsConnecting = types.BoolValue(n.IsConnecting)
	if n.LastStatusChange != nil {
		p.LastStatusChange = types.StringValue(*n.LastStatusChange)
	} else {
		p.LastStatusChange = types.StringNull()
	}
	if n.LastStatusMessage != nil {
		p.LastStatusMessage = types.StringValue(*n.LastStatusMessage)
	} else {
		p.LastStatusMessage = types.StringNull()
	}
	p.IsTrafficTrackingActive = types.BoolValue(n.IsTrafficTrackingActive)
	p.UsersOnline = types.Int64Value(int64(n.UsersOnline))
	p.ViewPosition = types.Int64Value(int64(n.ViewPosition))
	p.XrayUptime = types.Float64Value(n.XrayUptime)
	p.ProviderDetails = rawJSONToString(n.Provider)
	p.System = rawJSONToString(n.System)
	p.Versions = rawJSONToString(n.Versions)
	p.CreatedAt = types.StringValue(n.CreatedAt)
	p.UpdatedAt = types.StringValue(n.UpdatedAt)

	if n.Port != nil {
		p.Port = types.Int64Value(int64(*n.Port))
	} else {
		p.Port = types.Int64Null()
	}
	if n.ProxyURL != nil {
		p.ProxyURL = types.StringValue(*n.ProxyURL)
	} else {
		p.ProxyURL = types.StringNull()
	}
	if n.CountryCode != "" {
		p.CountryCode = types.StringValue(n.CountryCode)
	}
	if n.Note != nil {
		p.Note = types.StringValue(*n.Note)
	} else {
		p.Note = types.StringNull()
	}
	if n.TrafficLimitBytes != nil {
		p.TrafficLimitBytes = types.Int64Value(*n.TrafficLimitBytes)
	} else {
		p.TrafficLimitBytes = types.Int64Null()
	}
	if n.TrafficUsedBytes != nil {
		p.TrafficUsedBytes = types.Int64Value(*n.TrafficUsedBytes)
	} else {
		p.TrafficUsedBytes = types.Int64Null()
	}
	if n.TrafficResetDay != nil {
		p.TrafficResetDay = types.Int64Value(int64(*n.TrafficResetDay))
	} else {
		p.TrafficResetDay = types.Int64Null()
	}
	if n.NotifyPercent != nil {
		p.NotifyPercent = types.Int64Value(int64(*n.NotifyPercent))
	} else {
		p.NotifyPercent = types.Int64Null()
	}
	if n.ConsumptionMultiplier != nil {
		p.ConsumptionMultiplier = types.Float64Value(*n.ConsumptionMultiplier)
	} else {
		p.ConsumptionMultiplier = types.Float64Null()
	}
	if n.NodeConsumptionMultiplier != nil {
		p.NodeConsumptionMultiplier = types.Float64Value(*n.NodeConsumptionMultiplier)
	} else {
		p.NodeConsumptionMultiplier = types.Float64Null()
	}
	p.Tags, _ = types.SetValueFrom(context.Background(), types.StringType, n.Tags)
	if n.ProviderUUID != nil {
		p.ProviderUUID = types.StringValue(*n.ProviderUUID)
	} else {
		p.ProviderUUID = types.StringNull()
	}
	if n.ActivePluginUUID != nil {
		p.ActivePluginUUID = types.StringValue(*n.ActivePluginUUID)
	} else {
		p.ActivePluginUUID = types.StringNull()
	}
	if n.ConfigProfile != nil {
		p.ConfigProfileUUID = types.StringValue(n.ConfigProfile.ActiveConfigProfileUUID)
		// inbounds as set
		elems := make([]attr.Value, 0, len(n.ConfigProfile.ActiveInbounds))
		for _, ib := range n.ConfigProfile.ActiveInbounds {
			elems = append(elems, types.StringValue(ib.UUID))
		}
		s, _ := types.SetValue(types.StringType, elems)
		p.ConfigProfileInbounds = s
	} else {
		p.ConfigProfileUUID = types.StringNull()
		p.ConfigProfileInbounds = types.SetNull(types.StringType)
	}
}
