package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Nodes Metrics Data Source (#23) ───

type nodesMetricsDataSource struct {
	client *Client
}

type statsItemModel struct {
	Tag      types.String `tfsdk:"tag"`
	Upload   types.String `tfsdk:"upload"`
	Download types.String `tfsdk:"download"`
}

type nodesMetricsItemModel struct {
	NodeUUID       types.String     `tfsdk:"node_uuid"`
	NodeName       types.String     `tfsdk:"node_name"`
	CountryEmoji   types.String     `tfsdk:"country_emoji"`
	ProviderName   types.String     `tfsdk:"provider_name"`
	UsersOnline    types.Int64      `tfsdk:"users_online"`
	InboundsStats  []statsItemModel `tfsdk:"inbounds_stats"`
	OutboundsStats []statsItemModel `tfsdk:"outbounds_stats"`
}

type nodesMetricsDataSourceModel struct {
	Nodes []nodesMetricsItemModel `tfsdk:"nodes"`
}

func NewNodesMetricsDataSource() datasource.DataSource {
	return &nodesMetricsDataSource{}
}

func (d *nodesMetricsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_nodes_metrics"
}

func (d *nodesMetricsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	statsNestedAttrs := map[string]schema.Attribute{
		"tag":      schema.StringAttribute{Computed: true},
		"upload":   schema.StringAttribute{Computed: true},
		"download": schema.StringAttribute{Computed: true},
	}

	resp.Schema = schema.Schema{
		Description: "Returns per-node metrics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"nodes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"node_uuid":     schema.StringAttribute{Computed: true},
						"node_name":     schema.StringAttribute{Computed: true},
						"country_emoji": schema.StringAttribute{Computed: true},
						"provider_name": schema.StringAttribute{Computed: true},
						"users_online":  schema.Int64Attribute{Computed: true},
						"inbounds_stats": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: statsNestedAttrs,
							},
						},
						"outbounds_stats": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: statsNestedAttrs,
							},
						},
					},
				},
			},
		},
	}
}

func (d *nodesMetricsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nodesMetricsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetNodesMetrics(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get nodes metrics", err.Error())
		return
	}

	var state nodesMetricsDataSourceModel

	rawNodes, ok := data["nodes"].([]any)
	if ok {
		for _, n := range rawNodes {
			nm, ok := n.(map[string]any)
			if !ok {
				continue
			}
			item := nodesMetricsItemModel{
				NodeUUID:     types.StringValue(getString(nm["nodeUuid"])),
				NodeName:     types.StringValue(getString(nm["nodeName"])),
				CountryEmoji: types.StringValue(getString(nm["countryEmoji"])),
				ProviderName: types.StringValue(getString(nm["providerName"])),
				UsersOnline:  types.Int64Value(int64(getNumber(nm["usersOnline"]))),
			}

			item.InboundsStats = parseStatsList(nm["inboundsStats"])
			item.OutboundsStats = parseStatsList(nm["outboundsStats"])

			state.Nodes = append(state.Nodes, item)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseStatsList(raw any) []statsItemModel {
	items, ok := raw.([]any)
	if !ok {
		return []statsItemModel{}
	}
	var result []statsItemModel
	for _, s := range items {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, statsItemModel{
			Tag:      types.StringValue(getString(sm["tag"])),
			Upload:   types.StringValue(getString(sm["upload"])),
			Download: types.StringValue(getString(sm["download"])),
		})
	}
	if result == nil {
		result = []statsItemModel{}
	}
	return result
}

// getString extracts a string from an interface{}, returning "" if nil.
func getString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
