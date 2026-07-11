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

type hostResource struct {
	client *Client
}

type hostResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Remark          types.String `tfsdk:"remark"`
	Address         types.String `tfsdk:"address"`
	Port            types.Int64  `tfsdk:"port"`
	SNI             types.String `tfsdk:"sni"`
	HostHeader      types.String `tfsdk:"host_header"`
	ALPN            types.String `tfsdk:"alpn"`
	Fingerprint     types.String `tfsdk:"fingerprint"`
	IsDisabled      types.Bool   `tfsdk:"is_disabled"`
	SecurityLayer   types.String `tfsdk:"security_layer"`
	ServerDescription types.String `tfsdk:"server_description"`
	IsHidden        types.Bool   `tfsdk:"is_hidden"`
	ShuffleHost     types.Bool   `tfsdk:"shuffle_host"`
	// Inbound link
	ConfigProfileUUID        types.String `tfsdk:"config_profile_uuid"`
	ConfigProfileInboundUUID types.String `tfsdk:"config_profile_inbound_uuid"`
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
			"server_description": schema.StringAttribute{
				Optional:    true,
				Description: "Short server description (max 30 chars).",
			},
			"is_hidden": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Hide host from subscription.",
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
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// ─── Conversions ───

func planToHost(p *hostResourceModel) *Host {
	h := &Host{
		UUID:          p.UUID.ValueString(),
		Remark:        p.Remark.ValueString(),
		Address:       p.Address.ValueString(),
		Port:          int(p.Port.ValueInt64()),
		IsDisabled:    p.IsDisabled.ValueBool(),
		SecurityLayer: p.SecurityLayer.ValueString(),
		IsHidden:      p.IsHidden.ValueBool(),
		ShuffleHost:   p.ShuffleHost.ValueBool(),
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
	h.Inbound = &HostInbound{
		ConfigProfileUUID:        p.ConfigProfileUUID.ValueString(),
		ConfigProfileInboundUUID: p.ConfigProfileInboundUUID.ValueString(),
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
	if h.Inbound != nil {
		p.ConfigProfileUUID = types.StringValue(h.Inbound.ConfigProfileUUID)
		p.ConfigProfileInboundUUID = types.StringValue(h.Inbound.ConfigProfileInboundUUID)
	}
}

// Ensure attr import is used
var _ attr.Type = types.StringType
