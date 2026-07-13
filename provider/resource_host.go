package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type hostResource struct {
	client *Client
}

type hostResourceModel struct {
	UUID                   types.String `tfsdk:"uuid"`
	Remark                 types.String `tfsdk:"remark"`
	Address                types.String `tfsdk:"address"`
	Port                   types.Int64  `tfsdk:"port"`
	SNI                    types.String `tfsdk:"sni"`
	HostHeader             types.String `tfsdk:"host_header"`
	ALPN                   types.String `tfsdk:"alpn"`
	Fingerprint            types.String `tfsdk:"fingerprint"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
	SecurityLayer          types.String `tfsdk:"security_layer"`
	XHTTPExtraParams       types.String `tfsdk:"xhttp_extra_params"`
	MuxParams              types.String `tfsdk:"mux_params"`
	SockoptParams          types.String `tfsdk:"sockopt_params"`
	FinalMask              types.String `tfsdk:"final_mask"`
	ServerDescription      types.String `tfsdk:"server_description"`
	IsHidden               types.Bool   `tfsdk:"is_hidden"`
	OverrideSniFromAddress types.Bool   `tfsdk:"override_sni_from_address"`
	KeepSniBlank           types.Bool   `tfsdk:"keep_sni_blank"`
	PinnedPeerCertSha256   types.String `tfsdk:"pinned_peer_cert_sha256"`
	VerifyPeerCertByName   types.String `tfsdk:"verify_peer_cert_by_name"`
	VlessRouteID           types.Int64  `tfsdk:"vless_route_id"`
	ShuffleHost            types.Bool   `tfsdk:"shuffle_host"`
	// Inbound link
	ConfigProfileUUID            types.String `tfsdk:"config_profile_uuid"`
	ConfigProfileInboundUUID     types.String `tfsdk:"config_profile_inbound_uuid"`
	Tags                         types.List   `tfsdk:"tags"`
	Nodes                        types.List   `tfsdk:"nodes"`
	MihomoX25519                 types.Bool   `tfsdk:"mihomo_x25519"`
	MihomoIPVersion              types.String `tfsdk:"mihomo_ip_version"`
	XrayJSONTemplateUUID         types.String `tfsdk:"xray_json_template_uuid"`
	ExcludedInternalSquads       types.List   `tfsdk:"excluded_internal_squads"`
	ExcludeFromSubscriptionTypes types.Set    `tfsdk:"exclude_from_subscription_types"`
	Path                         types.String `tfsdk:"path"`
}

func NewHostResource() resource.Resource {
	return &hostResource{}
}

func (r *hostResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_host"
}

func (r *hostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave host (connection endpoint for VPN users).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "UUID of the host (assigned by the panel).",
			},
			"remark": schema.StringAttribute{
				Required:    true,
				Description: "Display name for the host (max 40 chars).",
			},
			"address": schema.StringAttribute{
				Required:    true,
				Description: "Server address (IP or hostname).",
			},
			"port": schema.Int64Attribute{
				Required:    true,
				Description: "Server port.",
			},
			"sni": schema.StringAttribute{
				Optional:    true,
				Description: "TLS Server Name Indication.",
			},
			"host_header": schema.StringAttribute{
				Optional:    true,
				Description: "Host header for HTTP/WebSocket.",
			},
			"alpn": schema.StringAttribute{
				Optional:    true,
				Description: "ALPN value (h3, h2, http/1.1, or combinations).",
			},
			"fingerprint": schema.StringAttribute{
				Optional:    true,
				Description: "TLS fingerprint (e.g. chrome, firefox).",
			},
			"is_disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the host is disabled.",
			},
			"security_layer": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Security layer: DEFAULT, TLS, or NONE.",
			},
			"xhttp_extra_params": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "XHTTP extra parameters as JSON."},
			"mux_params":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Mux parameters as JSON."},
			"sockopt_params":     schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Socket options as JSON."},
			"final_mask":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{canonicalJSONPlanModifier{}}, Description: "Final mask configuration as JSON."},
			"server_description": schema.StringAttribute{
				Optional:    true,
				Description: "Short server description (max 30 chars).",
			},
			"is_hidden": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Hide host from subscription.",
			},
			"override_sni_from_address": schema.BoolAttribute{
				Optional: true, Computed: true, Description: "Derive SNI from the host address.",
			},
			"keep_sni_blank": schema.BoolAttribute{
				Optional: true, Computed: true, Description: "Keep SNI blank instead of deriving it.",
			},
			"pinned_peer_cert_sha256": schema.StringAttribute{
				Optional: true, Description: "Pinned peer certificate SHA-256 value.",
			},
			"verify_peer_cert_by_name": schema.StringAttribute{
				Optional: true, Description: "Peer certificate name to verify.",
			},
			"vless_route_id": schema.Int64Attribute{
				Optional: true, Description: "VLESS route ID (0-65535).",
			},
			"shuffle_host": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Shuffle host order in subscription.",
			},
			"config_profile_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the config profile this host belongs to.",
			},
			"config_profile_inbound_uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the inbound within the config profile.",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of tags (max 10, uppercase letters/numbers/underscores/colons, max 36 chars each).",
			},
			"nodes": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of node UUIDs associated with this host.",
			},
			"mihomo_x25519": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable Mihomo X25519 proxy.",
			},
			"mihomo_ip_version": schema.StringAttribute{
				Optional: true, Description: "Mihomo IP version preference.",
			},
			"xray_json_template_uuid": schema.StringAttribute{
				Optional: true, Description: "Xray JSON subscription template UUID.",
			},
			"excluded_internal_squads": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Internal squad UUIDs from which this host is excluded.",
			},
			"exclude_from_subscription_types": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Subscription template types from which this host is excluded.",
			},
			"path": schema.StringAttribute{
				Optional:    true,
				Description: "WebSocket path or HTTP path.",
			},
		},
	}
}

func (r *hostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := planToHost(&plan)
	created, err := r.client.CreateHost(ctx, host)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create host", err.Error())
		return
	}

	hostToPlan(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		return
	}

	host, err := r.client.GetHostByUUID(ctx, uuid)
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "host not found, removing from state", map[string]any{"uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read host", err.Error())
		return
	}

	hostToPlan(host, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := planToHost(&plan)
	host.UUID = plan.UUID.ValueString()

	updated, err := r.client.UpdateHost(ctx, host)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update host", err.Error())
		return
	}

	hostToPlan(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if err := r.client.DeleteHost(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Failed to delete host", err.Error())
		return
	}
}

func (r *hostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

// ─── Conversions ───

func planToHost(p *hostResourceModel) *Host {
	h := &Host{
		UUID:                   p.UUID.ValueString(),
		Remark:                 p.Remark.ValueString(),
		Address:                p.Address.ValueString(),
		Port:                   int(p.Port.ValueInt64()),
		IsDisabled:             p.IsDisabled.ValueBool(),
		SecurityLayer:          p.SecurityLayer.ValueString(),
		IsHidden:               p.IsHidden.ValueBool(),
		OverrideSniFromAddress: p.OverrideSniFromAddress.ValueBool(),
		KeepSniBlank:           p.KeepSniBlank.ValueBool(),
		ShuffleHost:            p.ShuffleHost.ValueBool(),
	}
	if !p.SNI.IsNull() {
		sni := p.SNI.ValueString()
		h.SNI = &sni
	}
	if !p.HostHeader.IsNull() {
		hh := p.HostHeader.ValueString()
		h.HostHeader = &hh
	}
	if !p.ALPN.IsNull() {
		alpn := p.ALPN.ValueString()
		h.ALPN = &alpn
	}
	if !p.Fingerprint.IsNull() {
		fp := p.Fingerprint.ValueString()
		h.Fingerprint = &fp
	}
	if !p.ServerDescription.IsNull() {
		sd := p.ServerDescription.ValueString()
		h.ServerDescription = &sd
	}
	if !p.XHTTPExtraParams.IsNull() && !p.XHTTPExtraParams.IsUnknown() {
		h.XHTTPExtraParams = json.RawMessage(p.XHTTPExtraParams.ValueString())
	}
	if !p.MuxParams.IsNull() && !p.MuxParams.IsUnknown() {
		h.MuxParams = json.RawMessage(p.MuxParams.ValueString())
	}
	if !p.SockoptParams.IsNull() && !p.SockoptParams.IsUnknown() {
		h.SockoptParams = json.RawMessage(p.SockoptParams.ValueString())
	}
	if !p.FinalMask.IsNull() && !p.FinalMask.IsUnknown() {
		h.FinalMask = json.RawMessage(p.FinalMask.ValueString())
	}
	if !p.PinnedPeerCertSha256.IsNull() && !p.PinnedPeerCertSha256.IsUnknown() {
		value := p.PinnedPeerCertSha256.ValueString()
		h.PinnedPeerCertSha256 = &value
	}
	if !p.VerifyPeerCertByName.IsNull() && !p.VerifyPeerCertByName.IsUnknown() {
		value := p.VerifyPeerCertByName.ValueString()
		h.VerifyPeerCertByName = &value
	}
	if !p.VlessRouteID.IsNull() && !p.VlessRouteID.IsUnknown() {
		value := int(p.VlessRouteID.ValueInt64())
		h.VlessRouteID = &value
	}
	h.Inbound = &HostInbound{
		ConfigProfileUUID:        p.ConfigProfileUUID.ValueString(),
		ConfigProfileInboundUUID: p.ConfigProfileInboundUUID.ValueString(),
	}
	if !p.Tags.IsNull() {
		tags := []string{}
		for _, v := range p.Tags.Elements() {
			tags = append(tags, v.(types.String).ValueString())
		}
		h.Tags = tags
	}
	if !p.Nodes.IsNull() {
		nodes := []string{}
		for _, v := range p.Nodes.Elements() {
			nodes = append(nodes, v.(types.String).ValueString())
		}
		h.Nodes = nodes
	}
	if !p.MihomoX25519.IsNull() {
		h.MihomoX25519 = p.MihomoX25519.ValueBool()
	}
	if !p.MihomoIPVersion.IsNull() && !p.MihomoIPVersion.IsUnknown() {
		value := p.MihomoIPVersion.ValueString()
		h.MihomoIPVersion = &value
	}
	if !p.XrayJSONTemplateUUID.IsNull() && !p.XrayJSONTemplateUUID.IsUnknown() {
		value := p.XrayJSONTemplateUUID.ValueString()
		h.XrayJsonTemplateUUID = &value
	}
	if !p.ExcludedInternalSquads.IsNull() {
		squads := []string{}
		for _, v := range p.ExcludedInternalSquads.Elements() {
			squads = append(squads, v.(types.String).ValueString())
		}
		h.ExcludedInternalSquads = squads
	}
	if !p.ExcludeFromSubscriptionTypes.IsNull() && !p.ExcludeFromSubscriptionTypes.IsUnknown() {
		for _, value := range p.ExcludeFromSubscriptionTypes.Elements() {
			h.ExcludeFromSubscriptionTypes = append(h.ExcludeFromSubscriptionTypes, value.(types.String).ValueString())
		}
	}
	if !p.Path.IsNull() {
		pathVal := p.Path.ValueString()
		h.Path = &pathVal
	}
	return h
}

func hostToPlan(h *Host, p *hostResourceModel) {
	p.UUID = types.StringValue(h.UUID)
	p.Remark = types.StringValue(h.Remark)
	p.Address = types.StringValue(h.Address)
	p.Port = types.Int64Value(int64(h.Port))
	p.IsDisabled = types.BoolValue(h.IsDisabled)
	p.IsHidden = types.BoolValue(h.IsHidden)
	p.OverrideSniFromAddress = types.BoolValue(h.OverrideSniFromAddress)
	p.KeepSniBlank = types.BoolValue(h.KeepSniBlank)
	p.ShuffleHost = types.BoolValue(h.ShuffleHost)

	if h.SecurityLayer != "" {
		p.SecurityLayer = types.StringValue(h.SecurityLayer)
	}
	if h.SNI != nil {
		p.SNI = types.StringValue(*h.SNI)
	} else {
		p.SNI = types.StringNull()
	}
	if h.HostHeader != nil {
		p.HostHeader = types.StringValue(*h.HostHeader)
	} else {
		p.HostHeader = types.StringNull()
	}
	if h.ALPN != nil {
		p.ALPN = types.StringValue(*h.ALPN)
	} else {
		p.ALPN = types.StringNull()
	}
	if h.Fingerprint != nil {
		p.Fingerprint = types.StringValue(*h.Fingerprint)
	} else {
		p.Fingerprint = types.StringNull()
	}
	if h.ServerDescription != nil {
		p.ServerDescription = types.StringValue(*h.ServerDescription)
	} else {
		p.ServerDescription = types.StringNull()
	}
	p.XHTTPExtraParams = rawJSONToString(h.XHTTPExtraParams)
	p.MuxParams = rawJSONToString(h.MuxParams)
	p.SockoptParams = rawJSONToString(h.SockoptParams)
	p.FinalMask = rawJSONToString(h.FinalMask)
	if h.PinnedPeerCertSha256 != nil {
		p.PinnedPeerCertSha256 = types.StringValue(*h.PinnedPeerCertSha256)
	} else {
		p.PinnedPeerCertSha256 = types.StringNull()
	}
	if h.VerifyPeerCertByName != nil {
		p.VerifyPeerCertByName = types.StringValue(*h.VerifyPeerCertByName)
	} else {
		p.VerifyPeerCertByName = types.StringNull()
	}
	if h.VlessRouteID != nil {
		p.VlessRouteID = types.Int64Value(int64(*h.VlessRouteID))
	} else {
		p.VlessRouteID = types.Int64Null()
	}
	if h.Inbound != nil {
		p.ConfigProfileUUID = types.StringValue(h.Inbound.ConfigProfileUUID)
		p.ConfigProfileInboundUUID = types.StringValue(h.Inbound.ConfigProfileInboundUUID)
	}
	if h.Tags != nil {
		elems := make([]attr.Value, 0, len(h.Tags))
		for _, t := range h.Tags {
			elems = append(elems, types.StringValue(t))
		}
		tagsList, _ := types.ListValue(types.StringType, elems)
		p.Tags = tagsList
	}
	if h.Nodes != nil {
		elems := make([]attr.Value, 0, len(h.Nodes))
		for _, n := range h.Nodes {
			elems = append(elems, types.StringValue(n))
		}
		nodesList, _ := types.ListValue(types.StringType, elems)
		p.Nodes = nodesList
	}
	p.MihomoX25519 = types.BoolValue(h.MihomoX25519)
	if h.MihomoIPVersion != nil {
		p.MihomoIPVersion = types.StringValue(*h.MihomoIPVersion)
	} else {
		p.MihomoIPVersion = types.StringNull()
	}
	if h.XrayJsonTemplateUUID != nil {
		p.XrayJSONTemplateUUID = types.StringValue(*h.XrayJsonTemplateUUID)
	} else {
		p.XrayJSONTemplateUUID = types.StringNull()
	}
	if h.ExcludedInternalSquads != nil {
		elems := make([]attr.Value, 0, len(h.ExcludedInternalSquads))
		for _, s := range h.ExcludedInternalSquads {
			elems = append(elems, types.StringValue(s))
		}
		squadsList, _ := types.ListValue(types.StringType, elems)
		p.ExcludedInternalSquads = squadsList
	}
	p.ExcludeFromSubscriptionTypes, _ = types.SetValueFrom(context.Background(), types.StringType, h.ExcludeFromSubscriptionTypes)
	if h.Path != nil {
		p.Path = types.StringValue(*h.Path)
	}
}

// Ensure attr import is used
var _ attr.Type = types.StringType
