package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── System Recap Data Source (#22) ───

type systemRecapDataSource struct {
	client *Client
}

type systemRecapDataSourceModel struct {
	ThisMonthUsers         types.Int64  `tfsdk:"this_month_users"`
	ThisMonthTraffic       types.String `tfsdk:"this_month_traffic"`
	TotalUsers             types.Int64  `tfsdk:"total_users"`
	TotalNodes             types.Int64  `tfsdk:"total_nodes"`
	TotalTraffic           types.String `tfsdk:"total_traffic"`
	TotalNodesRam          types.String `tfsdk:"total_nodes_ram"`
	TotalNodesCpuCores     types.Int64  `tfsdk:"total_nodes_cpu_cores"`
	TotalDistinctCountries types.Int64  `tfsdk:"total_distinct_countries"`
	Version                types.String `tfsdk:"version"`
	InitDate               types.String `tfsdk:"init_date"`
}

func NewSystemRecapDataSource() datasource.DataSource {
	return &systemRecapDataSource{}
}

func (d *systemRecapDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_system_recap"
}

func (d *systemRecapDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns a recap of system totals (users, nodes, traffic, version).",
		Attributes: map[string]schema.Attribute{
			"this_month_users":         schema.Int64Attribute{Computed: true},
			"this_month_traffic":       schema.StringAttribute{Computed: true},
			"total_users":              schema.Int64Attribute{Computed: true},
			"total_nodes":              schema.Int64Attribute{Computed: true},
			"total_traffic":            schema.StringAttribute{Computed: true},
			"total_nodes_ram":          schema.StringAttribute{Computed: true},
			"total_nodes_cpu_cores":    schema.Int64Attribute{Computed: true},
			"total_distinct_countries": schema.Int64Attribute{Computed: true},
			"version":                  schema.StringAttribute{Computed: true},
			"init_date":                schema.StringAttribute{Computed: true},
		},
	}
}

func (d *systemRecapDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *systemRecapDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	data, err := d.client.GetSystemRecap(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system recap", err.Error())
		return
	}

	var state systemRecapDataSourceModel

	if thisMonth, ok := data["thisMonth"].(map[string]any); ok {
		state.ThisMonthUsers = types.Int64Value(int64(getNumber(thisMonth["users"])))
		if t, ok := thisMonth["traffic"].(string); ok {
			state.ThisMonthTraffic = types.StringValue(t)
		}
	}

	if total, ok := data["total"].(map[string]any); ok {
		state.TotalUsers = types.Int64Value(int64(getNumber(total["users"])))
		state.TotalNodes = types.Int64Value(int64(getNumber(total["nodes"])))
		state.TotalNodesCpuCores = types.Int64Value(int64(getNumber(total["nodesCpuCores"])))
		state.TotalDistinctCountries = types.Int64Value(int64(getNumber(total["distinctCountries"])))
		if t, ok := total["traffic"].(string); ok {
			state.TotalTraffic = types.StringValue(t)
		}
		if r, ok := total["nodesRam"].(string); ok {
			state.TotalNodesRam = types.StringValue(r)
		}
	}

	if v, ok := data["version"].(string); ok {
		state.Version = types.StringValue(v)
	}
	if d2, ok := data["initDate"].(string); ok {
		state.InitDate = types.StringValue(d2)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
