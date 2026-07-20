package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Passkeys Data Source ───

type passkeysDataSource struct {
	client *Client
}

type passkeysDataSourceModel struct {
	Passkeys []passkeyItem `tfsdk:"passkeys"`
}

type passkeyItem struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewPasskeysDataSource() datasource.DataSource {
	return &passkeysDataSource{}
}

func (d *passkeysDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_passkeys"
}

func (d *passkeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all WebAuthn passkeys registered for the current admin user. Requires admin JWT auth (username/password), not an API token.",
		Attributes: map[string]schema.Attribute{
			"passkeys": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{Computed: true, Description: "Passkey UUID."},
						"name":       schema.StringAttribute{Computed: true, Description: "Human-readable passkey name."},
						"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
					},
				},
			},
		},
	}
}

func (d *passkeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *passkeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	passkeys, err := d.client.GetAllPasskeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list passkeys", err.Error())
		return
	}

	var state passkeysDataSourceModel
	for _, p := range passkeys {
		item := passkeyItem{
			UUID:      types.StringValue(p.ID),
			Name:      types.StringValue(p.Name),
			CreatedAt: types.StringValue(p.CreatedAt),
		}
		state.Passkeys = append(state.Passkeys, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
