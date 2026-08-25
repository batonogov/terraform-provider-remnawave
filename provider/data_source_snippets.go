package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type snippetsDataSource struct{ client *Client }

type snippetsDataSourceModel struct {
	Total    types.Int64             `tfsdk:"total"`
	Snippets []snippetDataSourceItem `tfsdk:"snippets"`
}

type snippetDataSourceItem struct {
	Name    types.String `tfsdk:"name"`
	Snippet types.String `tfsdk:"snippet"`
}

func NewSnippetsDataSource() datasource.DataSource { return &snippetsDataSource{} }

func (d *snippetsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_snippets"
}

func (d *snippetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave Xray config snippets.",
		Attributes: map[string]schema.Attribute{
			"total": schema.Int64Attribute{Computed: true, Description: "Number of snippets reported by the panel."},
			"snippets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{Computed: true, Description: "Snippet name."},
					// Kept as a JSON string for the same reason remnawave_snippet
					// does: the payload is an opaque Xray fragment whose shape the
					// provider does not model.
					"snippet": schema.StringAttribute{Computed: true, Description: "Snippet content as a JSON string."},
				}},
			},
		},
	}
}

func (d *snippetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *snippetsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.GetSnippets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list snippets", err.Error())
		return
	}
	items := make([]snippetDataSourceItem, 0, len(result.Snippets))
	for _, snippet := range result.Snippets {
		encoded, err := json.Marshal(snippet.Snippet)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal snippet", err.Error())
			return
		}
		items = append(items, snippetDataSourceItem{
			Name:    types.StringValue(snippet.Name),
			Snippet: types.StringValue(string(encoded)),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &snippetsDataSourceModel{
		Total:    types.Int64Value(int64(result.Total)),
		Snippets: items,
	})...)
}
