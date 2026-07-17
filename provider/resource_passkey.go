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

// passkeyResource is an import-only resource that can delete passkeys.
// Passkeys cannot be created via Terraform because registration is an
// interactive WebAuthn ceremony. This resource supports:
//   - Import:   bring an existing passkey into state
//   - Delete:   destroy the passkey via DELETE /api/passkeys/:id
//   - Read:     verify the passkey still exists (list lookup)
type passkeyResource struct {
	client *Client
}

type passkeyModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewPasskeyResource() resource.Resource { return &passkeyResource{} }

func (r *passkeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_passkey"
}

func (r *passkeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave WebAuthn passkey. This is an import-only resource: passkeys cannot be created via Terraform (registration requires an interactive WebAuthn ceremony). Import an existing passkey by its UUID, then destroy it with `terraform destroy`. Requires admin JWT auth (username/password), not an API token.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "Passkey UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable passkey name.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp.",
			},
		},
	}
}

func (r *passkeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create is a no-op for import-only resources — this should never be called
// in practice. Terraform will error if someone tries `terraform apply` without
// importing first because all attributes are Computed.
func (r *passkeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Passkeys cannot be created via Terraform",
		"WebAuthn passkey registration is interactive and cannot be automated. "+
			"Import an existing passkey instead: terraform import remnawave_passkey.example <uuid>",
	)
}

func (r *passkeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state passkeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	passkeys, err := r.client.GetAllPasskeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read passkeys", err.Error())
		return
	}

	for _, p := range passkeys {
		if p.UUID != state.UUID.ValueString() {
			continue
		}
		state.Name = types.StringValue(p.Name)
		state.CreatedAt = types.StringValue(p.CreatedAt)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// Passkey no longer exists — remove from state
	resp.State.RemoveResource(ctx)
}

// Update is not supported — all attributes are Computed.
func (r *passkeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Forward computed values from plan to state; nothing to send to the API.
	var plan passkeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *passkeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state passkeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePasskey(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete passkey", err.Error())
	}
}

func (r *passkeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
