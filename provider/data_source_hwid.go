package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── HWID Stats Data Source ───

type hwidStatsDataSource struct {
	client *Client
}

type hwidStatsDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewHwidStatsDataSource() datasource.DataSource {
	return &hwidStatsDataSource{}
}

func (d *hwidStatsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_hwid_stats"
}

func (d *hwidStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns HWID statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the panel's HWID stats endpoint.",
			},
		},
	}
}

func (d *hwidStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *hwidStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	stats, err := d.client.GetHwidStats(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get HWID stats", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal HWID stats response", fmt.Sprintf("error: %s", err))
		return
	}

	state := hwidStatsDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── HWID Top Users Data Source ───

type hwidTopUsersDataSource struct {
	client *Client
}

type hwidTopUsersDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewHwidTopUsersDataSource() datasource.DataSource {
	return &hwidTopUsersDataSource{}
}

func (d *hwidTopUsersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_hwid_top_users"
}

func (d *hwidTopUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns top users by HWID device count from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the panel's HWID top-users endpoint.",
			},
		},
	}
}

func (d *hwidTopUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *hwidTopUsersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	topUsers, err := d.client.GetHwidTopUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get HWID top users", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(topUsers)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal HWID top users response", fmt.Sprintf("error: %s", err))
		return
	}

	state := hwidTopUsersDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
