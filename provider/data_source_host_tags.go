package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Host Tags Data Source ───

type hostTagsDataSource struct {
	client *Client
}

type hostTagsDataSourceModel struct {
	Tags types.List `tfsdk:"tags"`
}

func NewHostTagsDataSource() datasource.DataSource {
	return &hostTagsDataSource{}
}

func (d *hostTagsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_host_tags"
}

func (d *hostTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all unique host tags from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"tags": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Set of all unique tags assigned to any host.",
			},
		},
	}
}

func (d *hostTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *hostTagsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tags, err := d.client.GetHostTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list host tags", err.Error())
		return
	}

	var state hostTagsDataSourceModel
	state.Tags, _ = types.ListValueFrom(ctx, types.StringType, tags)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
