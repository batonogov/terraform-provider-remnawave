package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── External Squads Data Source ───

type externalSquadsDataSource struct {
	client *Client
}

type externalSquadsDataSourceModel struct {
	ExternalSquads []externalSquadDSItem `tfsdk:"external_squads"`
}

type externalSquadDSItem struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
}

func NewExternalSquadsDataSource() datasource.DataSource {
	return &externalSquadsDataSource{}
}

func (d *externalSquadsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_external_squads"
}

func (d *externalSquadsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave external squads. External squads are managed by name; use this " +
			"data source to discover existing squad UUIDs for brownfield import.",
		Attributes: map[string]schema.Attribute{
			"external_squads": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true, Description: "External squad UUID."},
						"name": schema.StringAttribute{Computed: true, Description: "External squad name."},
					},
				},
			},
		},
	}
}

func (d *externalSquadsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *externalSquadsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	squads, err := d.client.GetAllExternalSquads(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list external squads", err.Error())
		return
	}

	items := make([]externalSquadDSItem, len(squads))
	for i, s := range squads {
		items[i] = externalSquadDSItem{
			UUID: types.StringValue(s.UUID),
			Name: types.StringValue(s.Name),
		}
	}

	state := externalSquadsDataSourceModel{ExternalSquads: items}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
