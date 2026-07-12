package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── System Stats Data Source (#21) ───

type systemStatsDataSource struct {
	client *Client
}

type systemStatsDataSourceModel struct {
	Tz types.String `tfsdk:"tz"`

	CpuCores    types.Int64 `tfsdk:"cpu_cores"`
	MemoryTotal types.Int64 `tfsdk:"memory_total"`
	MemoryFree  types.Int64 `tfsdk:"memory_free"`
	MemoryUsed  types.Int64 `tfsdk:"memory_used"`
	Uptime      types.Int64 `tfsdk:"uptime"`
	Timestamp   types.Int64 `tfsdk:"timestamp"`

	UsersStatusCounts types.Map   `tfsdk:"users_status_counts"`
	UsersTotal        types.Int64 `tfsdk:"users_total"`

	OnlineLastDay  types.Int64 `tfsdk:"online_last_day"`
	OnlineLastWeek types.Int64 `tfsdk:"online_last_week"`
	OnlineNever    types.Int64 `tfsdk:"online_never"`
	OnlineNow      types.Int64 `tfsdk:"online_now"`

	NodesTotalOnline        types.Int64  `tfsdk:"nodes_total_online"`
	NodesTotalBytesLifetime types.String `tfsdk:"nodes_total_bytes_lifetime"`
}

func NewSystemStatsDataSource() datasource.DataSource {
	return &systemStatsDataSource{}
}

func (d *systemStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_system_stats"
}

func (d *systemStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns system statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"tz": schema.StringAttribute{
				Optional:    true,
				Description: "Timezone for the stats (e.g. 'Europe/Berlin'). If omitted, the server default is used.",
			},
			"cpu_cores":                  schema.Int64Attribute{Computed: true},
			"memory_total":               schema.Int64Attribute{Computed: true},
			"memory_free":                schema.Int64Attribute{Computed: true},
			"memory_used":                schema.Int64Attribute{Computed: true},
			"uptime":                     schema.Int64Attribute{Computed: true},
			"timestamp":                  schema.Int64Attribute{Computed: true},
			"users_status_counts":        schema.MapAttribute{Computed: true, ElementType: types.Int64Type},
			"users_total":                schema.Int64Attribute{Computed: true},
			"online_last_day":            schema.Int64Attribute{Computed: true},
			"online_last_week":           schema.Int64Attribute{Computed: true},
			"online_never":               schema.Int64Attribute{Computed: true},
			"online_now":                 schema.Int64Attribute{Computed: true},
			"nodes_total_online":         schema.Int64Attribute{Computed: true},
			"nodes_total_bytes_lifetime": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *systemStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *systemStatsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config systemStatsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tz := ""
	if !config.Tz.IsNull() && !config.Tz.IsUnknown() {
		tz = config.Tz.ValueString()
	}

	data, err := d.client.GetSystemStats(ctx, tz)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system stats", err.Error())
		return
	}

	var state systemStatsDataSourceModel
	state.Tz = config.Tz

	// cpu
	if cpu, ok := data["cpu"].(map[string]any); ok {
		state.CpuCores = types.Int64Value(int64(getNumber(cpu["cores"])))
	}

	// memory
	if mem, ok := data["memory"].(map[string]any); ok {
		state.MemoryTotal = types.Int64Value(int64(getNumber(mem["total"])))
		state.MemoryFree = types.Int64Value(int64(getNumber(mem["free"])))
		state.MemoryUsed = types.Int64Value(int64(getNumber(mem["used"])))
	}

	// uptime / timestamp
	state.Uptime = types.Int64Value(int64(getNumber(data["uptime"])))
	state.Timestamp = types.Int64Value(int64(getNumber(data["timestamp"])))

	// users
	if users, ok := data["users"].(map[string]any); ok {
		state.UsersTotal = types.Int64Value(int64(getNumber(users["totalUsers"])))
		if sc, ok := users["statusCounts"].(map[string]any); ok {
			m := make(map[string]int64, len(sc))
			for k, v := range sc {
				m[k] = int64(getNumber(v))
			}
			mv, diags := types.MapValueFrom(ctx, types.Int64Type, m)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			state.UsersStatusCounts = mv
		}
	}

	// online stats
	if online, ok := data["onlineStats"].(map[string]any); ok {
		state.OnlineLastDay = types.Int64Value(int64(getNumber(online["lastDay"])))
		state.OnlineLastWeek = types.Int64Value(int64(getNumber(online["lastWeek"])))
		state.OnlineNever = types.Int64Value(int64(getNumber(online["neverOnline"])))
		state.OnlineNow = types.Int64Value(int64(getNumber(online["onlineNow"])))
	}

	// nodes
	if nodes, ok := data["nodes"].(map[string]any); ok {
		state.NodesTotalOnline = types.Int64Value(int64(getNumber(nodes["totalOnline"])))
		if bytes, ok := nodes["totalBytesLifetime"].(string); ok {
			state.NodesTotalBytesLifetime = types.StringValue(bytes)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// getNumber extracts a number from an interface{} that may be float64 or int.
func getNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case fmt.Stringer:
		var f float64
		_, err := fmt.Sscanf(n.String(), "%f", &f)
		if err == nil {
			return f
		}
	}
	return 0
}
