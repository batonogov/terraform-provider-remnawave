package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type nodeResource struct {
	client *Client
}

type nodeResourceModel struct {
	UUID                    types.String `tfsdk:"uuid"`
	Name                    types.String `tfsdk:"name"`
	Address                 types.String `tfsdk:"address"`
	Port                    types.Int64  `tfsdk:"port"`
	IsTrafficTrackingActive types.Bool   `tfsdk:"is_traffic_tracking_active"`
	TrafficLimitBytes       types.Int64  `tfsdk:"traffic_limit_bytes"`
	TrafficResetDay         types.Int64  `tfsdk:"traffic_reset_day"`
	NotifyPercent           types.Int64  `tfsdk:"notify_percent"`
	CountryCode             types.String `tfsdk:"country_code"`
	IsConnected             types.Bool   `tfsdk:"is_connected"`
	IsDisabled              types.Bool   `tfsdk:"is_disabled"`
	IsConnecting            types.Bool   `tfsdk:"is_connecting"`
	UsersOnline             types.Int64  `tfsdk:"users_online"`
	Note                    types.String `tfsdk:"note"`
	// config_profile_uuid + config_profile_inbounds are required for create
	ConfigProfileUUID         types.String  `tfsdk:"config_profile_uuid"`
	ConfigProfileInbounds     types.Set     `tfsdk:"config_profile_inbounds"`
	ProxyURL                  types.String  `tfsdk:"proxy_url"`
	ConsumptionMultiplier     types.Float64 `tfsdk:"consumption_multiplier"`
	NodeConsumptionMultiplier types.Float64 `tfsdk:"node_consumption_multiplier"`
	Tags                      types.List    `tfsdk:"tags"`
	ProviderUUID              types.String  `tfsdk:"provider_uuid"`
	ActivePluginUUID          types.String  `tfsdk:"active_plugin_uuid"`
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
			},
			"is_traffic_tracking_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable traffic tracking for this node.",
			},
			"traffic_limit_bytes": schema.Int64Attribute{
				Optional:    true,
				Description: "Traffic limit in bytes for this node.",
			},
			"traffic_reset_day": schema.Int64Attribute{
				Optional:    true,
				Description: "Day of month (1-31) to reset traffic counter.",
			},
			"notify_percent": schema.Int64Attribute{
				Optional:    true,
				Description: "Notify at this traffic usage percentage (0-100).",
			},
			"country_code": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ISO 3166-1 alpha-2 country code (2 chars).",
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
			"users_online": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of users currently online on this node.",
			},
			"note": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form note (max 255 chars).",
			},
			"config_profile_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "UUID of the config profile assigned to this node.",
			},
			"config_profile_inbounds": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of inbound UUIDs enabled for this node's config profile.",
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Description: "SOCKS5 proxy URL for node communication. Format: socks5://[user:pass@]host:port",
			},
			"consumption_multiplier": schema.Float64Attribute{
				Optional:    true,
				Description: "Traffic consumption multiplier (0.0-100.0, 1 decimal place).",
			},
			"node_consumption_multiplier": schema.Float64Attribute{
				Optional:    true,
				Description: "Node consumption multiplier (0.0-100.0, 1 decimal place).",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of tags (max 10, uppercase letters/numbers/underscores/colons, max 36 chars each).",
			},
			"provider_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "UUID of the infra billing provider for this node.",
			},
			"active_plugin_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "UUID of the active node plugin.",
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
	if !p.TrafficLimitBytes.IsNull() {
		tl := p.TrafficLimitBytes.ValueInt64()
		n.TrafficLimitBytes = &tl
	}
	if !p.TrafficResetDay.IsNull() {
		rd := int(p.TrafficResetDay.ValueInt64())
		n.TrafficResetDay = &rd
	}
	if !p.NotifyPercent.IsNull() {
		np := int(p.NotifyPercent.ValueInt64())
		n.NotifyPercent = &np
	}
	if !p.Note.IsNull() {
		note := p.Note.ValueString()
		n.Note = &note
	}
	if !p.ConfigProfileUUID.IsNull() && p.ConfigProfileUUID.ValueString() != "" {
		inbounds := []string{}
		if !p.ConfigProfileInbounds.IsNull() {
			for _, v := range p.ConfigProfileInbounds.Elements() {
				inbounds = append(inbounds, v.(types.String).ValueString())
			}
		}
		n.ConfigProfile = &NodeConfigProfile{
			ActiveConfigProfileUUID: p.ConfigProfileUUID.ValueString(),
			ActiveInbounds:          inbounds,
		}
	}
	if !p.ProxyURL.IsNull() {
		proxyURL := p.ProxyURL.ValueString()
		n.ProxyURL = &proxyURL
	}
	if !p.ConsumptionMultiplier.IsNull() {
		n.ConsumptionMultiplier = p.ConsumptionMultiplier.ValueFloat64()
	}
	if !p.NodeConsumptionMultiplier.IsNull() {
		n.NodeConsumptionMultiplier = p.NodeConsumptionMultiplier.ValueFloat64()
	}
	if !p.Tags.IsNull() {
		tags := []string{}
		for _, v := range p.Tags.Elements() {
			tags = append(tags, v.(types.String).ValueString())
		}
		n.Tags = tags
	}
	if !p.ProviderUUID.IsNull() {
		providerUUID := p.ProviderUUID.ValueString()
		n.ProviderUUID = &providerUUID
	}
	if !p.ActivePluginUUID.IsNull() {
		activePluginUUID := p.ActivePluginUUID.ValueString()
		n.ActivePluginUUID = &activePluginUUID
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
	p.IsTrafficTrackingActive = types.BoolValue(n.IsTrafficTrackingActive)
	p.UsersOnline = types.Int64Value(int64(n.UsersOnline))

	if n.Port != nil {
		p.Port = types.Int64Value(int64(*n.Port))
	}
	if n.CountryCode != "" {
		p.CountryCode = types.StringValue(n.CountryCode)
	}
	if n.Note != nil {
		p.Note = types.StringValue(*n.Note)
	}
	if n.TrafficLimitBytes != nil {
		p.TrafficLimitBytes = types.Int64Value(*n.TrafficLimitBytes)
	}
	if n.TrafficResetDay != nil {
		p.TrafficResetDay = types.Int64Value(int64(*n.TrafficResetDay))
	}
	if n.NotifyPercent != nil {
		p.NotifyPercent = types.Int64Value(int64(*n.NotifyPercent))
	}
	if n.ConfigProfile != nil {
		p.ConfigProfileUUID = types.StringValue(n.ConfigProfile.ActiveConfigProfileUUID)
		// inbounds as set
		elems := make([]attr.Value, 0, len(n.ConfigProfile.ActiveInbounds))
		for _, ib := range n.ConfigProfile.ActiveInbounds {
			elems = append(elems, types.StringValue(ib))
		}
		s, _ := types.SetValue(types.StringType, elems)
		p.ConfigProfileInbounds = s
	}
	if n.ProxyURL != nil {
		p.ProxyURL = types.StringValue(*n.ProxyURL)
	}
	p.ConsumptionMultiplier = types.Float64Value(n.ConsumptionMultiplier)
	p.NodeConsumptionMultiplier = types.Float64Value(n.NodeConsumptionMultiplier)
	if n.Tags != nil {
		elems := make([]attr.Value, 0, len(n.Tags))
		for _, t := range n.Tags {
			elems = append(elems, types.StringValue(t))
		}
		tagsList, _ := types.ListValue(types.StringType, elems)
		p.Tags = tagsList
	}
	if n.ProviderUUID != nil {
		p.ProviderUUID = types.StringValue(*n.ProviderUUID)
	}
	if n.ActivePluginUUID != nil {
		p.ActivePluginUUID = types.StringValue(*n.ActivePluginUUID)
	}
}
