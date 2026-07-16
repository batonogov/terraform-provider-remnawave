package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── User IPs Data Source ───

type userIPsDataSource struct {
	client *Client
}

type userIPsDataSourceModel struct {
	UUID types.String   `tfsdk:"uuid"`
	IPs  []types.String `tfsdk:"ips"`
}

func NewUserIPsDataSource() datasource.DataSource {
	return &userIPsDataSource{}
}

func (d *userIPsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_user_ips"
}

func (d *userIPsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of IPs that a user is currently connected from, via the Remnawave IP Control module. " +
			"This is an asynchronous operation (the panel queues a job and results are polled for).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the user to fetch connection IPs for.",
			},
			"ips": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of IP addresses the user is connected from.",
			},
		},
	}
}

func (d *userIPsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userIPsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ips, err := d.client.FetchUserIPs(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch user IPs", err.Error())
		return
	}

	for _, ip := range ips {
		config.IPs = append(config.IPs, types.StringValue(ip))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
