package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── Nodes Data Source ───

type nodesDataSource struct {
	client *Client
}

type nodesDataSourceModel struct {
	Nodes []nodeItem `tfsdk:"nodes"`
}

type nodeItem struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Address     types.String `tfsdk:"address"`
	Port        types.Int64  `tfsdk:"port"`
	CountryCode types.String `tfsdk:"country_code"`
	IsConnected types.Bool   `tfsdk:"is_connected"`
	IsDisabled  types.Bool   `tfsdk:"is_disabled"`
	UsersOnline types.Int64  `tfsdk:"users_online"`
}

func NewNodesDataSource() datasource.DataSource {
	return &nodesDataSource{}
}

func (d *nodesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_nodes"
}

func (d *nodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave nodes.",
		Attributes: map[string]schema.Attribute{
			"nodes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":         schema.StringAttribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"address":      schema.StringAttribute{Computed: true},
						"port":         schema.Int64Attribute{Computed: true},
						"country_code": schema.StringAttribute{Computed: true},
						"is_connected": schema.BoolAttribute{Computed: true},
						"is_disabled":  schema.BoolAttribute{Computed: true},
						"users_online": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *nodesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nodesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	nodes, err := d.client.GetAllNodes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list nodes", err.Error())
		return
	}

	var state nodesDataSourceModel
	for _, n := range nodes {
		item := nodeItem{
			UUID:        types.StringValue(n.UUID),
			Name:        types.StringValue(n.Name),
			Address:     types.StringValue(n.Address),
			CountryCode: types.StringValue(n.CountryCode),
			IsConnected: types.BoolValue(n.IsConnected),
			IsDisabled:  types.BoolValue(n.IsDisabled),
			UsersOnline: types.Int64Value(int64(n.UsersOnline)),
		}
		if n.Port != nil {
			item.Port = types.Int64Value(int64(*n.Port))
		}
		state.Nodes = append(state.Nodes, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── Users Data Source ───

type usersDataSource struct {
	client *Client
}

type usersDataSourceModel struct {
	Users []userItem `tfsdk:"users"`
}

type userItem struct {
	UUID     types.String `tfsdk:"uuid"`
	Username types.String `tfsdk:"username"`
	Status   types.String `tfsdk:"status"`
	Tag      types.String `tfsdk:"tag"`
}

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

func (d *usersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_users"
}

func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave users.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":     schema.StringAttribute{Computed: true},
						"username": schema.StringAttribute{Computed: true},
						"status":   schema.StringAttribute{Computed: true},
						"tag":      schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *usersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	users, err := d.client.GetAllUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list users", err.Error())
		return
	}

	var state usersDataSourceModel
	for _, u := range users {
		item := userItem{
			UUID:     types.StringValue(u.UUID),
			Username: types.StringValue(u.Username),
			Status:   types.StringValue(u.Status),
		}
		if u.Tag != nil {
			item.Tag = types.StringValue(*u.Tag)
		} else {
			item.Tag = types.StringNull()
		}
		state.Users = append(state.Users, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── Hosts Data Source ───

type hostsDataSource struct {
	client *Client
}

type hostsDataSourceModel struct {
	Hosts []hostItem `tfsdk:"hosts"`
}

type hostItem struct {
	UUID    types.String `tfsdk:"uuid"`
	Remark  types.String `tfsdk:"remark"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
}

func NewHostsDataSource() datasource.DataSource {
	return &hostsDataSource{}
}

func (d *hostsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_hosts"
}

func (d *hostsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave hosts.",
		Attributes: map[string]schema.Attribute{
			"hosts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":    schema.StringAttribute{Computed: true},
						"remark":  schema.StringAttribute{Computed: true},
						"address": schema.StringAttribute{Computed: true},
						"port":    schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *hostsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *hostsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	hosts, err := d.client.GetAllHosts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list hosts", err.Error())
		return
	}

	var state hostsDataSourceModel
	for _, h := range hosts {
		state.Hosts = append(state.Hosts, hostItem{
			UUID:    types.StringValue(h.UUID),
			Remark:  types.StringValue(h.Remark),
			Address: types.StringValue(h.Address),
			Port:    types.Int64Value(int64(h.Port)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── Config Profiles Data Source ───

type configProfilesDataSource struct {
	client *Client
}

type configProfilesDataSourceModel struct {
	ConfigProfiles []configProfileItem `tfsdk:"config_profiles"`
}

type configProfileItem struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
}

func NewConfigProfilesDataSource() datasource.DataSource {
	return &configProfilesDataSource{}
}

func (d *configProfilesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_config_profiles"
}

func (d *configProfilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Remnawave config profiles.",
		Attributes: map[string]schema.Attribute{
			"config_profiles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *configProfilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *configProfilesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	profiles, err := d.client.GetAllConfigProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list config profiles", err.Error())
		return
	}

	var state configProfilesDataSourceModel
	for _, p := range profiles {
		state.ConfigProfiles = append(state.ConfigProfiles, configProfileItem{
			UUID: types.StringValue(p.UUID),
			Name: types.StringValue(p.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── System Health Data Source ───

type systemHealthDataSource struct {
	client *Client
}

type systemHealthDataSourceModel struct {
	Response types.String `tfsdk:"response"`
}

func NewSystemHealthDataSource() datasource.DataSource {
	return &systemHealthDataSource{}
}

func (d *systemHealthDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "remnawave_system_health"
}

func (d *systemHealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns system health and statistics from the Remnawave panel.",
		Attributes: map[string]schema.Attribute{
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON response from the panel's health endpoint.",
			},
		},
	}
}

func (d *systemHealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *systemHealthDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	health, err := d.client.GetSystemHealth(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system health", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(health)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal health response", fmt.Sprintf("error: %s", err))
		return
	}

	state := systemHealthDataSourceModel{
		Response: types.StringValue(string(jsonBytes)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
