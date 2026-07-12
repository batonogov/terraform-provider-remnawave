package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
}

func NewApiTokenResource() resource.Resource { return &apiTokenResource{} }

func (r *apiTokenResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_api_token"
}

func (r *apiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave API token. Note: requires admin JWT auth (not API token). Token value only returned on create.",
		Attributes: map[string]schema.Attribute{
			"uuid":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: "Token UUID."},
			"name":            schema.StringAttribute{Required: true, Description: "Token name (2-30 chars)."},
			"expires_in_days": schema.Int64Attribute{Required: true, Description: "Token expiration in days."},
			"scopes":          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Token scopes (default: ['*'])."},
			"token":           schema.StringAttribute{Computed: true, Sensitive: true, Description: "JWT token value (only available on create)."},
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
	token := &ApiToken{Name: plan.Name.ValueString(), Scopes: []string{"*"}}
	if !plan.Scopes.IsNull() {
		token.Scopes = nil
		for _, v := range plan.Scopes.Elements() {
			token.Scopes = append(token.Scopes, v.(types.String).ValueString())
		}
	}
	created, err := r.client.CreateApiToken(ctx, &ApiToken{Name: plan.Name.ValueString()})
	_ = token
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API token", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	plan.Token = types.StringValue(created.Token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// API token list doesn't return the token value, so Read is essentially a no-op.
	// State is preserved from create.
}

func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API tokens don't support PATCH — recreate
	resp.Diagnostics.AddError("API tokens cannot be updated", "Destroy and recreate the token to change name or scopes.")
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
}
