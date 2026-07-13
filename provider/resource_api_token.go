package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type apiTokenResource struct{ client *Client }
type apiTokenModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Scopes        types.Set    `tfsdk:"scopes"`
	Token         types.String `tfsdk:"token"`
	ExpireAt      types.String `tfsdk:"expire_at"`
}

func NewApiTokenResource() resource.Resource { return &apiTokenResource{} }

func (r *apiTokenResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_api_token"
}

func (r *apiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave API token. Note: requires admin JWT auth (not API token). Token value only returned on create.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: "Token UUID."},
			"name": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			}, Description: "Token name (2-30 chars)."},
			"expires_in_days": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			}, Description: "Token expiration in days."},
			"scopes": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{
				setplanmodifier.RequiresReplace(),
			}, Description: "Token scopes (default: ['*'])."},
			"token":     schema.StringAttribute{Computed: true, Sensitive: true, Description: "JWT token value (only available on create)."},
			"expire_at": schema.StringAttribute{Computed: true, Description: "Token expiration timestamp returned by Remnawave."},
		},
	}
}

func (r *apiTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", "Expected *Client")
		return
	}
	r.client = client
}

func (r *apiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	token := &ApiToken{
		Name:          plan.Name.ValueString(),
		ExpiresInDays: plan.ExpiresInDays.ValueInt64(),
		Scopes:        []string{"*"},
	}
	if !plan.Scopes.IsNull() {
		token.Scopes = nil
		for _, v := range plan.Scopes.Elements() {
			token.Scopes = append(token.Scopes, v.(types.String).ValueString())
		}
	}
	created, err := r.client.CreateApiToken(ctx, token)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API token", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	plan.Token = types.StringValue(created.Token)
	plan.ExpireAt = types.StringValue(created.ExpireAt)
	plan.Scopes, resp.Diagnostics = types.SetValueFrom(ctx, types.StringType, created.Scopes)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokens, err := r.client.GetAllApiTokens(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read API tokens", err.Error())
		return
	}
	for _, token := range tokens {
		if token.UUID != state.UUID.ValueString() {
			continue
		}
		state.Name = types.StringValue(token.Name)
		state.ExpireAt = types.StringValue(token.ExpireAt)
		state.Scopes, resp.Diagnostics = types.SetValueFrom(ctx, types.StringType, token.Scopes)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected API token update", "All configurable API token attributes require replacement.")
}

func (r *apiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApiToken(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete API token", err.Error())
	}
}

func (r *apiTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected API token UUID, got %q", req.ID))
	}
}
