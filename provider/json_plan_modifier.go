package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// canonicalHeaderMapJSONPlanModifier validates a JSON header map while
// preserving configured casing. Remnawave 3.0 persists keys in lowercase, so
// resources reconcile that wire normalization when refreshing state.
type canonicalHeaderMapJSONPlanModifier struct{}

func (canonicalHeaderMapJSONPlanModifier) Description(context.Context) string {
	return "Normalizes a JSON header map and rejects case-insensitive duplicate names."
}

func (canonicalHeaderMapJSONPlanModifier) MarkdownDescription(ctx context.Context) string {
	return canonicalHeaderMapJSONPlanModifier{}.Description(ctx)
}

func (canonicalHeaderMapJSONPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.ValueString() == "" {
		return
	}
	if _, err := canonicalHeaderMapJSON(req.PlanValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid HTTP header map", err.Error())
		return
	}
	canonical, err := canonicalJSONString(req.PlanValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid HTTP header map", err.Error())
		return
	}
	resp.PlanValue = types.StringValue(canonical)
}

// canonicalHeaderListJSONPlanModifier validates a JSON header-name array while
// preserving configured casing.
type canonicalHeaderListJSONPlanModifier struct{}

func (canonicalHeaderListJSONPlanModifier) Description(context.Context) string {
	return "Normalizes a JSON header-name list."
}

func (canonicalHeaderListJSONPlanModifier) MarkdownDescription(ctx context.Context) string {
	return canonicalHeaderListJSONPlanModifier{}.Description(ctx)
}

func (canonicalHeaderListJSONPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.ValueString() == "" {
		return
	}
	if _, err := canonicalHeaderListJSON(req.PlanValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid HTTP header list", err.Error())
		return
	}
	canonical, err := canonicalJSONString(req.PlanValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid HTTP header list", err.Error())
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

func canonicalHeaderMapJSON(value string) (string, error) {
	var decoded map[string]string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", err
	}
	if decoded == nil {
		return "", fmt.Errorf("header map must be a JSON object")
	}
	normalized := make(map[string]string, len(decoded))
	originalNames := make(map[string]string, len(decoded))
	for name, headerValue := range decoded {
		lower := strings.ToLower(name)
		if previous, exists := originalNames[lower]; exists && previous != name {
			return "", fmt.Errorf("header names %q and %q differ only by case", previous, name)
		}
		originalNames[lower] = name
		normalized[lower] = headerValue
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func canonicalHeaderListJSON(value string) (string, error) {
	var decoded []string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", err
	}
	if decoded == nil {
		return "", fmt.Errorf("header list must be a JSON array")
	}
	for i := range decoded {
		decoded[i] = strings.ToLower(decoded[i])
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// preserveEquivalentHeaderMap keeps the configured/state spelling when the
// backend changed only header-name casing. A real key or value change still
// replaces state and is therefore visible as drift.
func preserveEquivalentHeaderMap(previous, current types.String) (types.String, error) {
	if previous.IsNull() || previous.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return current, nil
	}
	previousCanonical, err := canonicalHeaderMapJSON(previous.ValueString())
	if err != nil {
		return types.StringNull(), fmt.Errorf("normalize prior header map: %w", err)
	}
	currentCanonical, err := canonicalHeaderMapJSON(current.ValueString())
	if err != nil {
		return types.StringNull(), fmt.Errorf("normalize response header map: %w", err)
	}
	if previousCanonical == currentCanonical {
		return previous, nil
	}
	return current, nil
}

func preserveEquivalentHeaderList(previous, current types.String) (types.String, error) {
	if previous.IsNull() || previous.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return current, nil
	}
	previousCanonical, err := canonicalHeaderListJSON(previous.ValueString())
	if err != nil {
		return types.StringNull(), fmt.Errorf("normalize prior header list: %w", err)
	}
	currentCanonical, err := canonicalHeaderListJSON(current.ValueString())
	if err != nil {
		return types.StringNull(), fmt.Errorf("normalize response header list: %w", err)
	}
	if previousCanonical == currentCanonical {
		return previous, nil
	}
	return current, nil
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
		"egressFilter": {}, "connectionDrop": {}, "preStart": {},
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
