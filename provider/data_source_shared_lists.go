package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sharedListsDataSource struct{ client *Client }

type sharedListsDataSourceModel struct {
	SharedLists []sharedListDataSourceItem `tfsdk:"shared_lists"`
}

type sharedListDataSourceItem struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	ItemsCount types.Int64  `tfsdk:"items_count"`
}

func NewSharedListsDataSource() datasource.DataSource { return &sharedListsDataSource{} }

func (d *sharedListsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_shared_lists"
}

func (d *sharedListsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists previews of all global Remnawave 3.3+ node-plugin shared lists.",
		Attributes: map[string]schema.Attribute{
			"shared_lists": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name":        schema.StringAttribute{Computed: true, Description: "Shared-list name without the ext: prefix."},
					"type":        schema.StringAttribute{Computed: true, Description: "Shared-list type (ipList or asList)."},
					"items_count": schema.Int64Attribute{Computed: true, Description: "Number of entries in the list."},
				}},
			},
		},
	}
}

func (d *sharedListsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *sharedListsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if err := requireBackend3_3(ctx, d.client, "shared lists"); err != nil {
		resp.Diagnostics.AddError("Unsupported shared lists data source", err.Error())
		return
	}
	result, err := d.client.GetAllSharedLists(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list shared lists", err.Error())
		return
	}
	items := make([]sharedListDataSourceItem, 0, len(result.SharedLists))
	for _, list := range result.SharedLists {
		items = append(items, sharedListDataSourceItem{
			Name:       types.StringValue(list.Name),
			Type:       types.StringValue(list.Type),
			ItemsCount: types.Int64Value(list.ItemsCount),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &sharedListsDataSourceModel{SharedLists: items})...)
}
