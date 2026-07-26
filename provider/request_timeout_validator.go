package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = positiveDurationValidator{}

type positiveDurationValidator struct{}

func (positiveDurationValidator) Description(context.Context) string {
	return "value must be a valid Go duration greater than zero"
}

func (v positiveDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v positiveDurationValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := parsePositiveDuration(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid request_timeout",
			err.Error(),
		)
	}
}

func parsePositiveDuration(value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("must be a valid Go duration: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("duration must be greater than zero, got %q", value)
	}
	return duration, nil
}
