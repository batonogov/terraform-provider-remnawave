package provider

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

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
		Description: "Manages a Remnawave API token. Note: requires admin JWT auth (not API token). Token value only returned on create. Import supports compound ID `<uuid>,<expires_in_days>` to seed the required replacement attribute.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: "Token UUID."},
			"name": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			}, Description: "Token name (2-30 chars)."},
			"expires_in_days": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			}, Description: "Token expiration in days. On import, can be derived from createdAt/expireAt or provided via compound import ID `<uuid>,<expires_in_days>`."},
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
		// Derive expires_in_days from createdAt and expireAt if not already
		// in state (e.g. after UUID-only import on 2.8.x).
		if state.ExpiresInDays.IsNull() && token.ExpireAt != "" {
			if days := deriveExpiresInDays(token.CreatedAt, token.ExpireAt); days > 0 {
				state.ExpiresInDays = types.Int64Value(days)
			}
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.State.RemoveResource(ctx)
}

// deriveExpiresInDays computes the whole-day duration between createdAt and
// expireAt. Returns 0 if either timestamp is empty or cannot be parsed.
// Backend timestamps use ISO 8601 (e.g. 2025-01-15T10:30:00.000Z).
func deriveExpiresInDays(createdAt, expireAt string) int64 {
	if createdAt == "" || expireAt == "" {
		return 0
	}
	layout := time.RFC3339Nano
	// Try parsing; the backend uses .000Z suffix which RFC3339Nano handles.
	ct, err := time.Parse(layout, createdAt)
	if err != nil {
		// Fallback: try without sub-seconds
		ct, err = time.Parse("2006-01-02T15:04:05Z", createdAt)
		if err != nil {
			return 0
		}
	}
	et, err := time.Parse(layout, expireAt)
	if err != nil {
		et, err = time.Parse("2006-01-02T15:04:05Z", expireAt)
		if err != nil {
			return 0
		}
	}
	diff := et.Sub(ct)
	if diff <= 0 {
		return 0
	}
	// Round to nearest whole day to handle floating-point drift.
	return int64(math.Round(diff.Hours() / 24))
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
	// Support compound import ID: <uuid>,<expires_in_days>
	// The expires_in_days component is optional but recommended for 2.7.x
	// backends that don't return createdAt/expireAt.
	parts := strings.SplitN(req.ID, ",", 2)
	uuid := strings.TrimSpace(parts[0])
	if uuid == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected API token UUID or <uuid>,<expires_in_days>, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(uuid))...)
	if len(parts) == 2 {
		days, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || days <= 0 {
			resp.Diagnostics.AddError("Invalid expires_in_days in import ID", fmt.Sprintf("expected a positive integer, got %q", parts[1]))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("expires_in_days"), types.Int64Value(days))...)
	}
}
