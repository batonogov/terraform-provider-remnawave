package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// canonicalJSONPlanModifier normalizes configured JSON before apply so the
// state returned by Remnawave compares equal even when the configuration used
// different whitespace or object-key ordering.
type canonicalJSONPlanModifier struct{}

func (canonicalJSONPlanModifier) Description(context.Context) string {
	return "Normalizes JSON whitespace and object-key ordering."
}

func (canonicalJSONPlanModifier) MarkdownDescription(ctx context.Context) string {
	return canonicalJSONPlanModifier{}.Description(ctx)
}

func (canonicalJSONPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.ValueString() == "" {
		return
	}
	canonical, err := canonicalJSONString(req.PlanValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", err.Error())
		return
	}
	resp.PlanValue = types.StringValue(canonical)
}

// nodePluginJSONPlanModifier additionally materializes Remnawave's
// sharedLists default. Without this, the API adds the key after apply and the
// provider would return a value different from Terraform's known plan.
type nodePluginJSONPlanModifier struct{}

func (nodePluginJSONPlanModifier) Description(context.Context) string {
	return "Normalizes node plugin JSON and applies the sharedLists default."
}

func (nodePluginJSONPlanModifier) MarkdownDescription(ctx context.Context) string {
	return nodePluginJSONPlanModifier{}.Description(ctx)
}

func (nodePluginJSONPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.ValueString() == "" {
		return
	}
	canonical, _, err := canonicalNodePluginJSON(req.PlanValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid node plugin JSON", err.Error())
		return
	}
	resp.PlanValue = types.StringValue(canonical)
}

func canonicalJSONString(value string) (string, error) {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func canonicalNodePluginJSON(value string) (string, map[string]any, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", nil, err
	}
	if decoded == nil {
		return "", nil, fmt.Errorf("plugin_config must be a JSON object")
	}
	allowedKeys := map[string]struct{}{
		"sharedLists": {}, "torrentBlocker": {}, "ingressFilter": {},
		"egressFilter": {}, "connectionDrop": {},
	}
	for key := range decoded {
		if _, ok := allowedKeys[key]; !ok {
			return "", nil, fmt.Errorf("plugin_config contains unsupported key %q", key)
		}
	}
	if _, ok := decoded["sharedLists"]; !ok {
		decoded["sharedLists"] = []any{}
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", nil, err
	}
	return string(encoded), decoded, nil
}
