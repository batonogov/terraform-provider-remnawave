package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Bandwidth Stats (Nodes) Data Source ───

type bandwidthStatsDataSource struct {
	client *Client
}

type bandwidthStatsDataSourceModel struct {
	Start         types.String    `tfsdk:"start"`
	End           types.String    `tfsdk:"end"`
	TopNodesLimit types.Int64     `tfsdk:"top_nodes_limit"`
	Categories    []types.String  `tfsdk:"categories"`
	SparklineData []types.Float64 `tfsdk:"sparkline_data"`
	TopNodes      []bwTopNode     `tfsdk:"top_nodes"`
	Series        []bwSeries      `tfsdk:"series"`
}

type bwTopNode struct {
	UUID        types.String  `tfsdk:"uuid"`
	Color       types.String  `tfsdk:"color"`
	Name        types.String  `tfsdk:"name"`
	CountryCode types.String  `tfsdk:"country_code"`
	Total       types.Float64 `tfsdk:"total"`
}

type bwSeries struct {
	UUID        types.String    `tfsdk:"uuid"`
	Name        types.String    `tfsdk:"name"`
	Color       types.String    `tfsdk:"color"`
	CountryCode types.String    `tfsdk:"country_code"`
	Total       types.Float64   `tfsdk:"total"`
	Data        []types.Float64 `tfsdk:"data"`
}

func NewBandwidthStatsDataSource() datasource.DataSource {
	return &bandwidthStatsDataSource{}
}

func (d *bandwidthStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_bandwidth_stats"
}

func (d *bandwidthStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = bandwidthStatsSchema(false)
}

func (d *bandwidthStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *bandwidthStatsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config bandwidthStatsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	topNodesLimit := 0
	if !config.TopNodesLimit.IsNull() && !config.TopNodesLimit.IsUnknown() {
		topNodesLimit = int(config.TopNodesLimit.ValueInt64())
	}

	data, err := d.client.GetBandwidthStatsNodes(ctx, config.Start.ValueString(), config.End.ValueString(), topNodesLimit)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get bandwidth stats", err.Error())
		return
	}

	config.Categories = parseCategories(data)
	config.SparklineData = parseSparklineData(data)
	config.TopNodes = parseTopNodes(data)
	config.Series = parseSeries(data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ─── Bandwidth Stats (User) Data Source ───

type bandwidthStatsUserDataSource struct {
	client *Client
}

type bandwidthStatsUserDataSourceModel struct {
	UUID          types.String    `tfsdk:"uuid"`
	Start         types.String    `tfsdk:"start"`
	End           types.String    `tfsdk:"end"`
	TopNodesLimit types.Int64     `tfsdk:"top_nodes_limit"`
	Categories    []types.String  `tfsdk:"categories"`
	SparklineData []types.Float64 `tfsdk:"sparkline_data"`
	TopNodes      []bwTopNode     `tfsdk:"top_nodes"`
	Series        []bwSeries      `tfsdk:"series"`
}

func NewBandwidthStatsUserDataSource() datasource.DataSource {
	return &bandwidthStatsUserDataSource{}
}

func (d *bandwidthStatsUserDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_bandwidth_stats_user"
}

func (d *bandwidthStatsUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = bandwidthStatsSchema(true)
}

func (d *bandwidthStatsUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *bandwidthStatsUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config bandwidthStatsUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	topNodesLimit := 0
	if !config.TopNodesLimit.IsNull() && !config.TopNodesLimit.IsUnknown() {
		topNodesLimit = int(config.TopNodesLimit.ValueInt64())
	}

	data, err := d.client.GetBandwidthStatsUser(ctx, config.UUID.ValueString(), config.Start.ValueString(), config.End.ValueString(), topNodesLimit)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get bandwidth stats for user", err.Error())
		return
	}

	config.Categories = parseCategories(data)
	config.SparklineData = parseSparklineData(data)
	config.TopNodes = parseTopNodes(data)
	config.Series = parseSeries(data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ─── Shared schema & parsing helpers ───

func bandwidthStatsSchema(includeUUID bool) schema.Schema {
	attrs := map[string]schema.Attribute{
		"start": schema.StringAttribute{
			Required:    true,
			Description: "Start date in YYYY-MM-DD format.",
		},
		"end": schema.StringAttribute{
			Required:    true,
			Description: "End date in YYYY-MM-DD format.",
		},
		"top_nodes_limit": schema.Int64Attribute{
			Optional:    true,
			Description: "Maximum number of top nodes to return. Default: 20.",
		},
		"categories": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Date category labels.",
		},
		"sparkline_data": schema.ListAttribute{
			Computed:    true,
			ElementType: types.Float64Type,
			Description: "Aggregate bandwidth per category.",
		},
		"top_nodes": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"uuid":         schema.StringAttribute{Computed: true},
					"color":        schema.StringAttribute{Computed: true},
					"name":         schema.StringAttribute{Computed: true},
					"country_code": schema.StringAttribute{Computed: true},
					"total":        schema.Float64Attribute{Computed: true},
				},
			},
			Description: "Top nodes by bandwidth usage.",
		},
		"series": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"uuid":         schema.StringAttribute{Computed: true},
					"name":         schema.StringAttribute{Computed: true},
					"color":        schema.StringAttribute{Computed: true},
					"country_code": schema.StringAttribute{Computed: true},
					"total":        schema.Float64Attribute{Computed: true},
					"data": schema.ListAttribute{
						Computed:    true,
						ElementType: types.Float64Type,
					},
				},
			},
			Description: "Per-node bandwidth series.",
		},
	}

	if includeUUID {
		attrs["uuid"] = schema.StringAttribute{
			Required:    true,
			Description: "User UUID to fetch bandwidth stats for.",
		}
	}

	return schema.Schema{
		Description: "Returns bandwidth statistics from the Remnawave panel.",
		Attributes:  attrs,
	}
}

func parseCategories(data map[string]any) []types.String {
	var result []types.String
	if raw, ok := data["categories"].([]any); ok {
		for _, c := range raw {
			if s, ok := c.(string); ok {
				result = append(result, types.StringValue(s))
			}
		}
	}
	return result
}

func parseSparklineData(data map[string]any) []types.Float64 {
	var result []types.Float64
	if raw, ok := data["sparklineData"].([]any); ok {
		for _, v := range raw {
			if f, ok := v.(float64); ok {
				result = append(result, types.Float64Value(f))
			}
		}
	}
	return result
}

func parseTopNodes(data map[string]any) []bwTopNode {
	var result []bwTopNode
	if raw, ok := data["topNodes"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				result = append(result, parseBwTopNode(m))
			}
		}
	}
	return result
}

func parseSeries(data map[string]any) []bwSeries {
	var result []bwSeries
	if raw, ok := data["series"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				result = append(result, parseBwSeries(m))
			}
		}
	}
	return result
}

func parseBwTopNode(m map[string]any) bwTopNode {
	node := bwTopNode{}
	if v, ok := m["uuid"].(string); ok {
		node.UUID = types.StringValue(v)
	}
	if v, ok := m["color"].(string); ok {
		node.Color = types.StringValue(v)
	}
	if v, ok := m["name"].(string); ok {
		node.Name = types.StringValue(v)
	}
	if v, ok := m["countryCode"].(string); ok {
		node.CountryCode = types.StringValue(v)
	}
	if v, ok := m["total"].(float64); ok {
		node.Total = types.Float64Value(v)
	}
	return node
}

func parseBwSeries(m map[string]any) bwSeries {
	s := bwSeries{}
	if v, ok := m["uuid"].(string); ok {
		s.UUID = types.StringValue(v)
	}
	if v, ok := m["name"].(string); ok {
		s.Name = types.StringValue(v)
	}
	if v, ok := m["color"].(string); ok {
		s.Color = types.StringValue(v)
	}
	if v, ok := m["countryCode"].(string); ok {
		s.CountryCode = types.StringValue(v)
	}
	if v, ok := m["total"].(float64); ok {
		s.Total = types.Float64Value(v)
	}
	if raw, ok := m["data"].([]any); ok {
		for _, v := range raw {
			if f, ok := v.(float64); ok {
				s.Data = append(s.Data, types.Float64Value(f))
			}
		}
	}
	return s
}
