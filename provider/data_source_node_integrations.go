package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nodeIntegrationsDataSource struct{ client *Client }

type nodeIntegrationsDataSourceModel struct {
	NodeIntegrations []nodeIntegrationDataSourceItem `tfsdk:"node_integrations"`
}

type nodeIntegrationDataSourceItem struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Config      types.String `tfsdk:"config"`
}

func NewNodeIntegrationsDataSource() datasource.DataSource { return &nodeIntegrationsDataSource{} }

func (d *nodeIntegrationsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_node_integrations"
}

func (d *nodeIntegrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave 3.3+ node integrations.",
		Attributes: map[string]schema.Attribute{
			"node_integrations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"uuid":        schema.StringAttribute{Computed: true, Description: "Node integration UUID."},
					"name":        schema.StringAttribute{Computed: true, Description: "Node integration name."},
					"description": schema.StringAttribute{Computed: true, Description: "Optional node integration description."},
					"config":      schema.StringAttribute{Computed: true, Sensitive: true, Description: "Integration configuration as normalized JSON."},
				}},
			},
		},
	}
}

func (d *nodeIntegrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nodeIntegrationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if err := requireBackend3_3(ctx, d.client, "node integrations"); err != nil {
		resp.Diagnostics.AddError("Unsupported node integrations data source", err.Error())
		return
	}
	result, err := d.client.GetAllNodeIntegrations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list node integrations", err.Error())
		return
	}
	items := make([]nodeIntegrationDataSourceItem, 0, len(result.NodeIntegrations))
	for _, integration := range result.NodeIntegrations {
		config, err := json.Marshal(integration.Config)
		if err != nil {
			resp.Diagnostics.AddError("Failed to decode node integrations", fmt.Sprintf("marshal config for %s: %v", integration.UUID, err))
			return
		}
		item := nodeIntegrationDataSourceItem{
			UUID:   types.StringValue(integration.UUID),
			Name:   types.StringValue(integration.Name),
			Config: types.StringValue(string(config)),
		}
		if integration.Description != nil {
			item.Description = types.StringValue(*integration.Description)
		} else {
			item.Description = types.StringNull()
		}
		items = append(items, item)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &nodeIntegrationsDataSourceModel{NodeIntegrations: items})...)
}
