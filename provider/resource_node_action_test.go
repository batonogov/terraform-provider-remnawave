package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNodeActionResource_Restart(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-action-restart", "VLESS_NODE_ACTION_RESTART") + `
resource "remnawave_node" "test" {
  name                = "tf-acc-node-action-restart"
  address             = "127.0.0.10"
  port                = 2222
  config_profile_uuid = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

resource "remnawave_node_action" "test" {
  node_uuid     = remnawave_node.test.uuid
  action        = "restart"
  force_restart = true
  triggers      = ["init"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.test", "action", "restart"),
					resource.TestCheckResourceAttr("remnawave_node_action.test", "force_restart", "true"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "id"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "created_at"),
				),
			},
		},
	})
}

func TestAccNodeActionResource_ResetTraffic(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-action-reset", "VLESS_NODE_ACTION_RESET") + `
resource "remnawave_node" "test" {
  name                = "tf-acc-node-action-reset"
  address             = "127.0.0.10"
  port                = 2222
  config_profile_uuid = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

resource "remnawave_node_action" "test" {
  node_uuid = remnawave_node.test.uuid
  action    = "reset_traffic"
  triggers  = ["init"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_action.test", "action", "reset_traffic"),
					resource.TestCheckResourceAttr("remnawave_node_action.test", "force_restart", "false"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "id"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "created_at"),
				),
			},
		},
	})
}

// TestAccNodeActionResource_ResetTrafficAlias verifies that the legacy
// "reset-traffic" (hyphen) spelling still works as a backward-compatible
// alias and is normalized to the canonical "reset_traffic" form.
func TestAccNodeActionResource_ResetTrafficAlias(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-action-reset-alias", "VLESS_NODE_ACTION_RESET_ALIAS") + `
resource "remnawave_node" "test" {
  name                = "tf-acc-node-action-reset-alias"
  address             = "127.0.0.10"
  port                = 2222
  config_profile_uuid = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

resource "remnawave_node_action" "test" {
  node_uuid = remnawave_node.test.uuid
  action    = "reset-traffic"
  triggers  = ["init"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Alias is normalized to canonical "reset_traffic" in state.
					resource.TestCheckResourceAttr("remnawave_node_action.test", "action", "reset_traffic"),
					resource.TestCheckResourceAttr("remnawave_node_action.test", "force_restart", "false"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "id"),
					resource.TestCheckResourceAttrSet("remnawave_node_action.test", "created_at"),
				),
			},
		},
	})
}
