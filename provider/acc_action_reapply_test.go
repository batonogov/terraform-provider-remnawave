package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests cover the core semantics of action-as-resource resources
// (remnawave_user_action / remnawave_node_action): that changing the `triggers`
// list re-executes the imperative action on the backend. They also cover the
// previously-untested node_action enable/disable actions.
//
// Proof of re-execution differs per resource because the two resources expose
// different state:
//   - user_action has no execution timestamp in state, so re-execution is
//     proven via the backend `lastTrafficResetAt` field changing between the
//     first and second action invocation (server-side proof).
//   - node_action exposes a provider-set `created_at` (RFC3339, re-stamped on
//     every executeAction), so re-execution is proven by that timestamp
//     changing between steps (provider-side proof, with a short sleep to cross
//     a one-second boundary since RFC3339 has second precision).
//
// Both re-apply tests also assert that the new triggers value lands in state.

// TestAccUserAction_ReapplyWithChangedTrigger verifies that changing the
// triggers list re-executes the reset_traffic action on the user. After the
// first apply the backend lastTrafficResetAt is captured (T1); after the
// second apply (triggers changed) it must differ (T2) — proving the action
// re-ran server-side, not just that Terraform replaced the resource.
func TestAccUserAction_ReapplyWithChangedTrigger(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const userCfg = `
resource "remnawave_user" "test" {
  username            = "reapply-user-acc"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
`

	var userUUID, resetT1 string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Step 1: the action runs once with triggers = ["v1"].
				Config: providerCfg + userCfg + `
resource "remnawave_user_action" "reset" {
  user_uuid = remnawave_user.test.uuid
  action    = "reset_traffic"
  triggers  = ["v1"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.reset", "triggers.0", "v1"),
					testAccResourceUUID("remnawave_user.test", &userUUID),
					// Capture the backend traffic-reset timestamp produced by
					// the first action invocation so Step 2 can prove a re-run.
					func(*terraform.State) error {
						client := testAccBackendClient(t)
						user, err := client.GetUserByUUID(context.Background(), userUUID)
						if err != nil {
							return fmt.Errorf("read user for T1: %w", err)
						}
						if user.LastTrafficResetAt == nil || *user.LastTrafficResetAt == "" {
							return fmt.Errorf("expected lastTrafficResetAt to be set after reset_traffic, got %v", user.LastTrafficResetAt)
						}
						resetT1 = *user.LastTrafficResetAt
						return nil
					},
				),
			},
			{
				// Step 2: triggers change -> RequiresReplaceIfConfigured -> the
				// resource is recreated -> reset_traffic runs again.
				PreConfig: func() {
					// lastTrafficResetAt may carry sub-second precision; sleep
					// past a one-second boundary so a re-run is unambiguously
					// distinguishable from the previous timestamp.
					time.Sleep(1100 * time.Millisecond)
				},
				Config: providerCfg + userCfg + `
resource "remnawave_user_action" "reset" {
  user_uuid = remnawave_user.test.uuid
  action    = "reset_traffic"
  triggers  = ["v2"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.reset", "triggers.0", "v2"),
					func(*terraform.State) error {
						client := testAccBackendClient(t)
						user, err := client.GetUserByUUID(context.Background(), userUUID)
						if err != nil {
							return fmt.Errorf("read user for T2: %w", err)
						}
						if user.LastTrafficResetAt == nil || *user.LastTrafficResetAt == "" {
							return fmt.Errorf("expected lastTrafficResetAt to be set after re-apply, got %v", user.LastTrafficResetAt)
						}
						if *user.LastTrafficResetAt == resetT1 {
							return fmt.Errorf("lastTrafficResetAt did not change after triggers re-apply (action did not re-run server-side): before=%q after=%q", resetT1, *user.LastTrafficResetAt)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccNodeAction_ReapplyWithChangedTrigger verifies that changing the
// triggers list re-executes the reset_traffic action on the node. The node
// action resource re-stamps its provider-set created_at (RFC3339) on every
// executeAction, so re-execution is proven by created_at changing between the
// two steps (after a short sleep to cross a one-second boundary).
func TestAccNodeAction_ReapplyWithChangedTrigger(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const nodeBase = `
resource "remnawave_node" "test" {
  name                    = "tf-acc-node-reapply-action"
  address                 = "127.0.0.31"
  port                    = 2231
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`

	var createdAtT1 string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-reapply-action", "VLESS_NODE_REAPPLY_ACTION") + nodeBase + `
resource "remnawave_node_action" "reset" {
  node_uuid = remnawave_node.test.uuid
  action    = "reset_traffic"
  triggers  = ["v1"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.reset", "triggers.0", "v1"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.reset", "created_at"),
					func(s *terraform.State) error {
						createdAtT1 = s.RootModule().Resources["remnawave_node_action.reset"].Primary.Attributes["created_at"]
						return nil
					},
				),
			},
			{
				// triggers change -> the node_action Update path re-executes the
				// action (executeAction), re-stamping created_at.
				PreConfig: func() {
					// created_at is RFC3339 (second precision); sleep past a
					// one-second boundary so the re-stamp differs from T1.
					time.Sleep(1100 * time.Millisecond)
				},
				Config: providerCfg + testAccProfileConfig("node-reapply-action", "VLESS_NODE_REAPPLY_ACTION") + nodeBase + `
resource "remnawave_node_action" "reset" {
  node_uuid = remnawave_node.test.uuid
  action    = "reset_traffic"
  triggers  = ["v2"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.reset", "triggers.0", "v2"),
					func(s *terraform.State) error {
						createdT2 := s.RootModule().Resources["remnawave_node_action.reset"].Primary.Attributes["created_at"]
						if createdT2 == createdAtT1 {
							return fmt.Errorf("created_at did not change after triggers re-apply (action did not re-execute): before=%q after=%q", createdAtT1, createdT2)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccNodeAction_EnableDisable covers the previously-untested enable and
// disable node actions. It creates a node, disables it (verifying the backend
// reports isDisabled=true), then re-enables it (verifying isDisabled=false) so
// the node is left enabled. Both actions are verified by reading the backend
// directly (GetNodeByUUID) rather than the Terraform state: although the node
// resource exposes the computed `is_disabled` attribute, an independent backend
// read proves the server actually applied the action and is not just echoing
// what the provider wrote.
func TestAccNodeAction_EnableDisable(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const nodeBase = `
resource "remnawave_node" "test" {
  name                    = "tf-acc-node-enable-disable"
  address                 = "127.0.0.32"
  port                    = 2232
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`

	var nodeUUID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-enable-disable", "VLESS_NODE_ENABLE_DISABLE") + nodeBase + `
resource "remnawave_node_action" "disable" {
  node_uuid = remnawave_node.test.uuid
  action    = "disable"
  triggers  = ["once"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.disable", "action", "disable"),
					testAccResourceUUID("remnawave_node.test", &nodeUUID),
					func(*terraform.State) error {
						client := testAccBackendClient(t)
						node, err := client.GetNodeByUUID(context.Background(), nodeUUID)
						if err != nil {
							return fmt.Errorf("read node after disable: %w", err)
						}
						if !node.IsDisabled {
							return fmt.Errorf("expected node isDisabled=true after disable action, got false")
						}
						return nil
					},
				),
			},
			{
				Config: providerCfg + testAccProfileConfig("node-enable-disable", "VLESS_NODE_ENABLE_DISABLE") + nodeBase + `
resource "remnawave_node_action" "enable" {
  node_uuid = remnawave_node.test.uuid
  action    = "enable"
  triggers  = ["once"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.enable", "action", "enable"),
					func(*terraform.State) error {
						client := testAccBackendClient(t)
						node, err := client.GetNodeByUUID(context.Background(), nodeUUID)
						if err != nil {
							return fmt.Errorf("read node after enable: %w", err)
						}
						if node.IsDisabled {
							return fmt.Errorf("expected node isDisabled=false after enable action, got true")
						}
						return nil
					},
				),
			},
		},
	})
}
