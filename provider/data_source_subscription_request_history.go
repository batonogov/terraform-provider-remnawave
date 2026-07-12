package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Subscription Request History Data Source ───

type subscriptionRequestHistoryDataSource struct {
	client *Client
}

type subscriptionRequestHistoryDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewSubscriptionRequestHistoryDataSource() datasource.DataSource {
	return &subscriptionRequestHistoryDataSource{}
}

func (d *subscriptionRequestHistoryDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_subscription_request_history"
}

func (d *subscriptionRequestHistoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns subscription request history from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *subscriptionRequestHistoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subscriptionRequestHistoryDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.GetSubscriptionRequestHistory(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch subscription request history", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal subscription request history response", fmt.Sprintf("error: %s", err))
		return
	}

	state := subscriptionRequestHistoryDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
