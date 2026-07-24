package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/sync/errgroup"
)

// ─── Internal Squads Data Source ───

type internalSquadsDataSource struct {
	client *Client
}

type internalSquadsDataSourceModel struct {
	InternalSquads []internalSquadDSItem `tfsdk:"internal_squads"`
}

type internalSquadDSItem struct {
	UUID                types.String `tfsdk:"uuid"`
	Name                types.String `tfsdk:"name"`
	Inbounds            types.Set    `tfsdk:"inbounds"`
	AccessibleNodeUUIDs types.List   `tfsdk:"accessible_node_uuids"`
}

func NewInternalSquadsDataSource() datasource.DataSource {
	return &internalSquadsDataSource{}
}

func (d *internalSquadsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_internal_squads"
}

func (d *internalSquadsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave internal squads. Use this to discover existing squad UUIDs and " +
			"inbound assignments for brownfield import. Accessible node UUIDs are derived from each " +
			"squad's inbound configuration.",
		Attributes: map[string]schema.Attribute{
			"internal_squads": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true, Description: "Internal squad UUID."},
						"name": schema.StringAttribute{Computed: true, Description: "Internal squad name."},
						"inbounds": schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Set of config profile inbound UUIDs assigned to the squad.",
						},
						"accessible_node_uuids": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Read-only list of node UUIDs reachable through the squad's inbounds.",
						},
					},
				},
			},
		},
	}
}

func (d *internalSquadsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *internalSquadsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	squads, err := d.client.GetAllInternalSquads(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list internal squads", err.Error())
		return
	}

	items := make([]internalSquadDSItem, len(squads))
	for i, s := range squads {
		inboundElems := make([]attr.Value, 0, len(s.Inbounds))
		for _, ib := range s.Inbounds {
			inboundElems = append(inboundElems, types.StringValue(ib.UUID))
		}
		inboundsSet, _ := types.SetValue(types.StringType, inboundElems)

		items[i] = internalSquadDSItem{
			UUID:     types.StringValue(s.UUID),
			Name:     types.StringValue(s.Name),
			Inbounds: inboundsSet,
			// accessible_node_uuids is populated below (concurrent, fail-fast).
		}
	}

	// Fetch accessible nodes concurrently with bounded concurrency and fail-fast
	// semantics. Accessible nodes are NOT part of the list response, so one
	// additional request per squad is required. Using the errgroup-derived
	// context propagates cancellation (e.g. request_timeout) to all in-flight
	// calls. Each goroutine writes only to its own items[i] index.
	if len(items) > 0 {
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(5)
		for i := range items {
			squadUUID := squads[i].UUID
			idx := i
			g.Go(func() error {
				accessible, err := d.client.GetInternalSquadAccessibleNodes(gctx, squadUUID)
				if err != nil {
					return fmt.Errorf("accessible nodes for squad %s: %w", squadUUID, err)
				}
				nodeElems := make([]attr.Value, 0, len(accessible.AccessibleNodes))
				for _, n := range accessible.AccessibleNodes {
					nodeElems = append(nodeElems, types.StringValue(n.UUID))
				}
				nodeList, _ := types.ListValue(types.StringType, nodeElems)
				items[idx].AccessibleNodeUUIDs = nodeList
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			resp.Diagnostics.AddError("Failed to fetch accessible nodes for internal squads", err.Error())
			return
		}
	}

	state := internalSquadsDataSourceModel{InternalSquads: items}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
