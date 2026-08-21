package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type nodePluginResource struct{ client *Client }
type nodePluginModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	PluginConfig types.String `tfsdk:"plugin_config"`
}

func NewNodePluginResource() resource.Resource { return &nodePluginResource{} }

func (r *nodePluginResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "remnawave_node_plugin"
}

func (r *nodePluginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remnawave node plugin.",
		Attributes: map[string]schema.Attribute{
			"uuid":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":          schema.StringAttribute{Required: true, Description: "Plugin name (2-30 chars)."},
			"plugin_config": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{nodePluginJSONPlanModifier{}}, Description: "Plugin config as JSON. Supported keys are sharedLists, torrentBlocker, ingressFilter, egressFilter, connectionDrop, and preStart (Remnawave 3.1+). On Remnawave 3.3+, sharedLists is read as an effective compatibility view and omitted from plugin writes; manage global list contents with remnawave_shared_list. The torrentBlocker object accepts rulePlacement (0-1000) on Remnawave 3.3.1+ to position the injected routing rule; Remnawave 3.3.1 returns a default of 0 for that key, which the provider drops unless the configuration sets it."},
		},
	}
}

func (r *nodePluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodePluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nodePluginModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var pluginConfig map[string]any
	if !plan.PluginConfig.IsNull() && !plan.PluginConfig.IsUnknown() && plan.PluginConfig.ValueString() != "" {
		canonical, decoded, err := canonicalNodePluginJSON(plan.PluginConfig.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid plugin_config JSON", err.Error())
			return
		}
		plan.PluginConfig = types.StringValue(canonical)
		pluginConfig = decoded
	}
	if err := r.validatePluginConfigVersion(ctx, pluginConfig); err != nil {
		resp.Diagnostics.AddError("Unsupported node plugin configuration", err.Error())
		return
	}
	created, err := r.client.CreateNodePlugin(ctx, &NodePlugin{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create node plugin", err.Error())
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	rollback := func(summary string, cause error) {
		detail := cause.Error()
		if cleanupErr := r.client.DeleteNodePlugin(ctx, created.UUID); cleanupErr != nil {
			detail += fmt.Sprintf("; additionally failed to delete partially created plugin %s: %v", created.UUID, cleanupErr)
		}
		resp.Diagnostics.AddError(summary, detail)
	}
	switch {
	case pluginConfig != nil:
		updated, err := r.client.UpdateNodePlugin(ctx, &NodePlugin{UUID: created.UUID, Name: plan.Name.ValueString(), PluginConfig: pluginConfig})
		if err != nil {
			rollback("Failed to set plugin config", err)
			return
		}
		if updated.PluginConfig != nil {
			b, err := json.Marshal(alignNodePluginRulePlacement(pluginConfig, updated.PluginConfig))
			if err != nil {
				rollback("Failed to marshal plugin_config", err)
				return
			}
			plan.PluginConfig = types.StringValue(string(b))
		}
	case created.PluginConfig != nil:
		b, err := json.Marshal(created.PluginConfig)
		if err != nil {
			rollback("Failed to marshal plugin_config", err)
			return
		}
		plan.PluginConfig = types.StringValue(string(b))
	default:
		plan.PluginConfig = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodePluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nodePluginModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plugin, err := r.client.GetNodePluginByUUID(ctx, state.UUID.ValueString())
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "node plugin not found", map[string]any{"uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read node plugin", err.Error())
		return
	}
	state.UUID = types.StringValue(plugin.UUID)
	state.Name = types.StringValue(plugin.Name)
	if plugin.PluginConfig != nil {
		pluginConfig, err := r.pluginConfigForState(ctx, plugin.PluginConfig, state.PluginConfig)
		if err != nil {
			resp.Diagnostics.AddError("Failed to normalize plugin_config", err.Error())
			return
		}
		b, err := json.Marshal(pluginConfig)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal plugin_config", err.Error())
			return
		}
		state.PluginConfig = types.StringValue(string(b))
	} else {
		state.PluginConfig = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nodePluginResource) pluginConfigForState(ctx context.Context, remote any, previous types.String) (any, error) {
	is3_3, err := r.client.isVersionAtLeast3_3(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect node plugin contract: %w", err)
	}
	if !is3_3 {
		return remote, nil
	}

	remoteConfig, ok := remote.(map[string]any)
	if !ok || remoteConfig == nil {
		return remote, nil
	}
	normalized := make(map[string]any, len(remoteConfig))
	for key, value := range remoteConfig {
		if key != "sharedLists" {
			normalized[key] = value
		}
	}
	normalized["sharedLists"] = []any{}
	if previous.IsNull() || previous.IsUnknown() || previous.ValueString() == "" {
		return normalized, nil
	}
	_, previousConfig, err := canonicalNodePluginJSON(previous.ValueString())
	if err != nil {
		return nil, fmt.Errorf("normalize prior plugin_config: %w", err)
	}
	if sharedLists, exists := previousConfig["sharedLists"]; exists {
		normalized["sharedLists"] = sharedLists
	}
	return alignNodePluginRulePlacement(previousConfig, normalized), nil
}

func (r *nodePluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nodePluginModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plugin := &NodePlugin{UUID: plan.UUID.ValueString(), Name: plan.Name.ValueString()}
	var pluginConfig map[string]any
	if !plan.PluginConfig.IsNull() && plan.PluginConfig.ValueString() != "" {
		canonical, cfg, err := canonicalNodePluginJSON(plan.PluginConfig.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid plugin_config JSON", err.Error())
			return
		}
		plan.PluginConfig = types.StringValue(canonical)
		plugin.PluginConfig = cfg
		pluginConfig = cfg
	}
	if err := r.validatePluginConfigVersion(ctx, pluginConfig); err != nil {
		resp.Diagnostics.AddError("Unsupported node plugin configuration", err.Error())
		return
	}
	updated, err := r.client.UpdateNodePlugin(ctx, plugin)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update node plugin", err.Error())
		return
	}
	if updated.PluginConfig != nil {
		b, err := json.Marshal(alignNodePluginRulePlacement(pluginConfig, updated.PluginConfig))
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal plugin_config", err.Error())
			return
		}
		plan.PluginConfig = types.StringValue(string(b))
	} else {
		plan.PluginConfig = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nodePluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nodePluginModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNodePlugin(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete node plugin", err.Error())
	}
}

func (r *nodePluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), types.StringValue(req.ID))...)
}

func (r *nodePluginResource) validatePluginConfigVersion(ctx context.Context, pluginConfig map[string]any) error {
	if pluginConfig == nil {
		return nil
	}
	if _, ok := pluginConfig["preStart"]; ok {
		supported, err := r.client.isVersionAtLeast3_1(ctx)
		if err != nil {
			return fmt.Errorf("detect backend version for preStart: %w", err)
		}
		if !supported {
			return fmt.Errorf("preStart requires Remnawave 3.1 or newer")
		}
	}
	if nodePluginHasRulePlacement(pluginConfig) {
		supported, err := r.client.isVersionAtLeast3_3_1(ctx)
		if err != nil {
			return fmt.Errorf("detect backend version for torrentBlocker.rulePlacement: %w", err)
		}
		if !supported {
			return fmt.Errorf("torrentBlocker.rulePlacement requires Remnawave 3.3.1 or newer")
		}
	}
	return nil
}

// nodePluginHasRulePlacement reports whether a plugin config sets
// torrentBlocker.rulePlacement. Remnawave rejects nothing for the key on older
// panels: its schema is not strict, so 3.3.0 silently strips it instead.
func nodePluginHasRulePlacement(pluginConfig map[string]any) bool {
	torrentBlocker, ok := pluginConfig["torrentBlocker"].(map[string]any)
	if !ok {
		return false
	}
	_, exists := torrentBlocker["rulePlacement"]
	return exists
}

// alignNodePluginRulePlacement drops torrentBlocker.rulePlacement from a backend
// response when the configuration did not set it. Remnawave 3.3.1 applies a
// schema default of 0 to that key and stores the parsed config, so without this
// the value written to state would differ from Terraform's planned value.
// Remnawave 3.3.2 removed the default, which is why the provider must not
// materialize one of its own.
func alignNodePluginRulePlacement(configured map[string]any, remote any) any {
	remoteConfig, ok := remote.(map[string]any)
	if !ok || remoteConfig == nil {
		return remote
	}
	remoteBlocker, ok := remoteConfig["torrentBlocker"].(map[string]any)
	if !ok {
		return remote
	}
	if _, exists := remoteBlocker["rulePlacement"]; !exists {
		return remote
	}
	if configured != nil && nodePluginHasRulePlacement(configured) {
		return remote
	}
	torrentBlocker := make(map[string]any, len(remoteBlocker))
	for key, value := range remoteBlocker {
		if key != "rulePlacement" {
			torrentBlocker[key] = value
		}
	}
	normalized := make(map[string]any, len(remoteConfig))
	for key, value := range remoteConfig {
		normalized[key] = value
	}
	normalized["torrentBlocker"] = torrentBlocker
	return normalized
}
