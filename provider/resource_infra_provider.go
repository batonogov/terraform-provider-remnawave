package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type infraProviderResource struct{ client *Client }
type infraProviderModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	FaviconLink types.String `tfsdk:"favicon_link"`
	LoginURL    types.String `tfsdk:"login_url"`
}

func NewInfraProviderResource() resource.Resource { return &infraProviderResource{} }

func (r *infraProviderResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_infra_provider"
}

func (r *infraProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave infrastructure provider (for infra billing).",
		Attributes: map[string]schema.Attribute{
			"uuid":         schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":         schema.StringAttribute{Required: true, Description: "Provider name (2-30 chars)."},
			"favicon_link": schema.StringAttribute{Optional: true, Computed: true, Description: "Favicon URL."},
			"login_url":    schema.StringAttribute{Optional: true, Computed: true, Description: "Login URL."},
		},
	}
}

func (r *infraProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *infraProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan infraProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := &InfraProvider{Name: plan.Name.ValueString()}
	if !plan.FaviconLink.IsNull() && plan.FaviconLink.ValueString() != "" {
		f := plan.FaviconLink.ValueString()
		p.FaviconLink = &f
	}
	if !plan.LoginURL.IsNull() && plan.LoginURL.ValueString() != "" {
		l := plan.LoginURL.ValueString()
		p.LoginURL = &l
	}
	created, err := r.client.CreateInfraProvider(ctx, p)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create infra provider", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	plan.Name = types.StringValue(created.Name)
	if created.FaviconLink != nil {
		plan.FaviconLink = types.StringValue(*created.FaviconLink)
	} else {
		plan.FaviconLink = types.StringNull()
	}
	if created.LoginURL != nil {
		plan.LoginURL = types.StringValue(*created.LoginURL)
	} else {
		plan.LoginURL = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *infraProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state infraProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetInfraProviderByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "infra provider not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read infra provider", err.Error())
		return
	}
	state.UUID = types.StringValue(p.UUID)
	state.Name = types.StringValue(p.Name)
	if p.FaviconLink != nil {
		state.FaviconLink = types.StringValue(*p.FaviconLink)
	}
	if p.LoginURL != nil {
		state.LoginURL = types.StringValue(*p.LoginURL)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *infraProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan infraProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := &InfraProvider{UUID: plan.UUID.ValueString(), Name: plan.Name.ValueString()}
	if !plan.FaviconLink.IsNull() {
		f := plan.FaviconLink.ValueString()
		p.FaviconLink = &f
	}
	if !plan.LoginURL.IsNull() {
		l := plan.LoginURL.ValueString()
		p.LoginURL = &l
	}
	updated, err := r.client.UpdateInfraProvider(ctx, p)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update infra provider", err.Error())
		return
	}
	plan.UUID = types.StringValue(updated.UUID)
	plan.Name = types.StringValue(updated.Name)
	if updated.FaviconLink != nil {
		plan.FaviconLink = types.StringValue(*updated.FaviconLink)
	} else {
		plan.FaviconLink = types.StringNull()
	}
	if updated.LoginURL != nil {
		plan.LoginURL = types.StringValue(*updated.LoginURL)
	} else {
		plan.LoginURL = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *infraProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state infraProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInfraProvider(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete infra provider", err.Error())
	}
}

func (r *infraProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}
