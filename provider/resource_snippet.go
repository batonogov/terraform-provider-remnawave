package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type snippetResource struct{ client *Client }
type snippetModel struct {
	Name    types.String `tfsdk:"name"`
	Snippet types.String `tfsdk:"snippet"`
}

func NewSnippetResource() resource.Resource { return &snippetResource{} }

func (r *snippetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_snippet"
}

func (r *snippetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave Xray config snippet (keyed by name).",
		Attributes: map[string]schema.Attribute{
			"name":    schema.StringAttribute{Required: true, Description: "Snippet name (2-255 chars)."},
			"snippet": schema.StringAttribute{Required: true, Description: "Snippet content as JSON array string."},
		},
	}
}

func (r *snippetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	client, ok := req.ProviderData.(*Client)
	if !ok { resp.Diagnostics.AddError("Unexpected type", "Expected *Client"); return }
	r.client = client
}

func (r *snippetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan snippetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }
	var snippetData any
	if err := json.Unmarshal([]byte(plan.Snippet.ValueString()), &snippetData); err != nil {
		resp.Diagnostics.AddError("Invalid snippet JSON", err.Error()); return
	}
	_, err := r.client.CreateSnippet(ctx, &Snippet{Name: plan.Name.ValueString(), Snippet: snippetData})
	if err != nil { resp.Diagnostics.AddError("Failed to create snippet", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snippetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	// Snippets are returned as a list, find by name
	list, err := r.client.GetSnippets(ctx)
	if err != nil { resp.Diagnostics.AddError("Failed to read snippets", err.Error()); return }
	found := false
	for _, s := range list.Snippets {
		if s.Name != state.Name.ValueString() {
			continue
		}
		found = true
		b, err := json.Marshal(s.Snippet)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal snippet", err.Error())
			return
		}
		state.Snippet = types.StringValue(string(b))
		break
	}
	if !found { resp.State.RemoveResource(ctx); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *snippetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan snippetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }
	var snippetData any
	if err := json.Unmarshal([]byte(plan.Snippet.ValueString()), &snippetData); err != nil {
		resp.Diagnostics.AddError("Invalid snippet JSON", err.Error()); return
	}
	_, err := r.client.UpdateSnippet(ctx, &Snippet{Name: plan.Name.ValueString(), Snippet: snippetData})
	if err != nil { resp.Diagnostics.AddError("Failed to update snippet", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snippetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state snippetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.DeleteSnippet(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete snippet", err.Error())
	}
}

func (r *snippetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(req.ID))...)
}
