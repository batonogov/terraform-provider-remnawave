package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUserBulkActionValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		action  string
		days    types.Int64
		wantErr string
	}{
		{name: "reset traffic", action: "reset_traffic", days: types.Int64Null()},
		{name: "extend expiration", action: "extend_expiration", days: types.Int64Value(7)},
		{name: "extend missing days", action: "extend_expiration", days: types.Int64Null(), wantErr: "days is required"},
		{name: "extend days too small", action: "extend_expiration", days: types.Int64Value(0), wantErr: "days must be between"},
		{name: "extend days too large", action: "extend_expiration", days: types.Int64Value(10000), wantErr: "days must be between"},
		{name: "unknown action", action: "archive", days: types.Int64Null(), wantErr: "action must be one of"},
	}

	resource := &userBulkActionResource{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := &userBulkActionResourceModel{
				Action: types.StringValue(tt.action),
				UUIDs:  testStringList("user-1"),
				Days:   tt.days,
			}
			err := resource.validate(plan)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("validate() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}

	err := resource.validate(&userBulkActionResourceModel{
		Action: types.StringValue("reset_traffic"),
		UUIDs:  testStringList(),
		Days:   types.Int64Null(),
	})
	if err == nil || !strings.Contains(err.Error(), "uuids must contain") {
		t.Fatalf("validate() empty uuids error = %v", err)
	}
}

func TestBulkActionIDs(t *testing.T) {
	t.Parallel()

	userPlan := &userBulkActionResourceModel{
		Action: types.StringValue("extend_expiration"),
		UUIDs:  testStringList("user-1", "user-2"),
		Days:   types.Int64Value(7),
	}
	if got, want := userBulkActionID(userPlan), "extend_expiration:days=7:user-1:user-2"; got != want {
		t.Errorf("userBulkActionID() = %q, want %q", got, want)
	}

	nodePlan := &nodeBulkActionResourceModel{
		Action: types.StringValue("enable"),
		UUIDs:  testStringList("node-1", "node-2"),
	}
	if got, want := nodeBulkActionID(nodePlan), "enable:node-1:node-2"; got != want {
		t.Errorf("nodeBulkActionID() = %q, want %q", got, want)
	}

	hostPlan := &hostBulkActionResourceModel{
		Action: types.StringValue("disable"),
		UUIDs:  testStringList("host-1", "host-2"),
	}
	if got, want := bulkActionID(hostPlan), "disable:host-1:host-2"; got != want {
		t.Errorf("bulkActionID() = %q, want %q", got, want)
	}
}
