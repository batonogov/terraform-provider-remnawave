package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Bandwidth Realtime Data Source ───

type bandwidthRealtimeDataSource struct {
	client *Client
}

type bandwidthRealtimeDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewBandwidthRealtimeDataSource() datasource.DataSource {
	return &bandwidthRealtimeDataSource{}
}

func (d *bandwidthRealtimeDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_bandwidth_realtime"
}

func (d *bandwidthRealtimeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns realtime bandwidth statistics for all nodes from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *bandwidthRealtimeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *bandwidthRealtimeDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetBandwidthRealtime(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get bandwidth realtime stats", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal response", fmt.Sprintf("error: %s", err))
		return
	}

	state := bandwidthRealtimeDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── System Bandwidth Stats Data Source ───

type systemBandwidthStatsDataSource struct {
	client *Client
}

type systemBandwidthStatsDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewSystemBandwidthStatsDataSource() datasource.DataSource {
	return &systemBandwidthStatsDataSource{}
}

func (d *systemBandwidthStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_system_bandwidth_stats"
}

func (d *systemBandwidthStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns system bandwidth statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *systemBandwidthStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *systemBandwidthStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetSystemBandwidthStats(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system bandwidth stats", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal response", fmt.Sprintf("error: %s", err))
		return
	}

	state := systemBandwidthStatsDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── System Nodes Stats Data Source ───

type systemNodesStatsDataSource struct {
	client *Client
}

type systemNodesStatsDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewSystemNodesStatsDataSource() datasource.DataSource {
	return &systemNodesStatsDataSource{}
}

func (d *systemNodesStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_system_nodes_stats"
}

func (d *systemNodesStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns system nodes statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *systemNodesStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *systemNodesStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetSystemNodesStats(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system nodes stats", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal response", fmt.Sprintf("error: %s", err))
		return
	}

	state := systemNodesStatsDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── Subscription Request History Stats Data Source ───

type subscriptionRequestHistoryStatsDataSource struct {
	client *Client
}

type subscriptionRequestHistoryStatsDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewSubscriptionRequestHistoryStatsDataSource() datasource.DataSource {
	return &subscriptionRequestHistoryStatsDataSource{}
}

func (d *subscriptionRequestHistoryStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_subscription_request_history_stats"
}

func (d *subscriptionRequestHistoryStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns subscription request history statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *subscriptionRequestHistoryStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *subscriptionRequestHistoryStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetSubscriptionRequestHistoryStats(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get subscription request history stats", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal response", fmt.Sprintf("error: %s", err))
		return
	}

	state := subscriptionRequestHistoryStatsDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── Connection Keys Data Source ───

type connectionKeysDataSource struct {
	client *Client
}

type connectionKeysDataSourceModel struct {
	UUID     types.String `tfsdk:"uuid"`
	Response types.String `tfsdk:"response"`
}

func NewConnectionKeysDataSource() datasource.DataSource {
	return &connectionKeysDataSource{}
}

func (d *connectionKeysDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_connection_keys"
}

func (d *connectionKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns connection keys for a subscription from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the subscription.",
			},
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *connectionKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *connectionKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config connectionKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := d.client.GetConnectionKeys(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get connection keys", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal response", fmt.Sprintf("error: %s", err))
		return
	}

	state := connectionKeysDataSourceModel{
		UUID:     config.UUID,
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
