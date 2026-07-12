package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Subscriptions Data Source ───

type subscriptionsDataSource struct {
	client *Client
}

type subscriptionsDataSourceModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Username  types.String `tfsdk:"username"`
	ShortUUID types.String `tfsdk:"short_uuid"`
	Response  types.String `tfsdk:"response"`
}

func NewSubscriptionsDataSource() datasource.DataSource {
	return &subscriptionsDataSource{}
}

func (d *subscriptionsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_subscriptions"
}

func (d *subscriptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a subscription by UUID, username, or short UUID. Exactly one of uuid, username, or short_uuid must be provided.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Optional:    true,
				Description: "UUID of the subscription to look up.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Username of the subscription to look up.",
			},
			"short_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "Short UUID of the subscription to look up.",
			},
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the API.",
			},
		},
	}
}

func (d *subscriptionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subscriptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subscriptionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one of uuid / username / short_uuid is set.
	setCount := 0
	if !state.UUID.IsNull() && state.UUID.ValueString() != "" {
		setCount++
	}
	if !state.Username.IsNull() && state.Username.ValueString() != "" {
		setCount++
	}
	if !state.ShortUUID.IsNull() && state.ShortUUID.ValueString() != "" {
		setCount++
	}
	if setCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid combination of arguments",
			"Exactly one of uuid, username, or short_uuid must be set.",
		)
		return
	}

	var result map[string]any
	var err error

	switch {
	case !state.UUID.IsNull() && state.UUID.ValueString() != "":
		result, err = d.client.GetSubscriptionByUUID(ctx, state.UUID.ValueString())
	case !state.Username.IsNull() && state.Username.ValueString() != "":
		result, err = d.client.GetSubscriptionByUsername(ctx, state.Username.ValueString())
	case !state.ShortUUID.IsNull() && state.ShortUUID.ValueString() != "":
		result, err = d.client.GetSubscriptionByShortUUID(ctx, state.ShortUUID.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch subscription", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal subscription response", fmt.Sprintf("error: %s", err))
		return
	}

	state.Response = types.StringValue(string(jsonBytes))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
