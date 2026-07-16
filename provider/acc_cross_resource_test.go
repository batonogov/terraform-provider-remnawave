package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFullNodeStack verifies a full stack of interdependent resources:
// config_profile → node (with inbounds) + subscription_template → host.
func TestAccFullNodeStack(t *testing.T) {
	testAccPreCheck(t)
	// Provider bug: host tags stay unknown on 2.7.x after apply.
	if isBackend2_7() {
		t.Skip("host tags computed field is unknown on 2.7.x — provider bug")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("stack-profile", "VLESS_TCP_STACK_ACC") + `
resource "remnawave_subscription_template" "stack" {
  name          = "stack-template"
  template_type = "XRAY_JSON"
}
resource "remnawave_node" "stack" {
  name                    = "stack-node"
  address                 = "10.0.0.1"
  port                    = 443
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
resource "remnawave_host" "stack" {
  remark                      = "stack-host"
  address                     = "stack.example.com"
  port                        = 443
  security_layer              = "TLS"
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
  xray_json_template_uuid     = remnawave_subscription_template.stack.uuid
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_config_profile.profile", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_subscription_template.stack", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node.stack", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node.stack", "config_profile_uuid"),
				resource.TestCheckResourceAttr("remnawave_node.stack", "config_profile_inbounds.#", "1"),
				resource.TestCheckResourceAttrSet("remnawave_host.stack", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_host.stack", "xray_json_template_uuid"),
				resource.TestCheckResourceAttrSet("remnawave_host.stack", "config_profile_uuid"),
				resource.TestCheckResourceAttrSet("remnawave_host.stack", "config_profile_inbound_uuid"),
			),
		}},
	})
}

// TestAccBillingChain verifies the billing dependency chain:
// infra_provider + node → billing_node, and infra_provider → billing_history.
func TestAccBillingChain(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("billing-chain-profile", "VLESS_TCP_BILL_CHAIN_ACC") + `
resource "remnawave_infra_provider" "chain" {
  name = "billing-chain-provider"
}
resource "remnawave_node" "chain" {
  name                    = "billing-chain-node"
  address                 = "10.30.40.50"
  port                    = 6666
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
resource "remnawave_billing_node" "chain" {
  provider_uuid   = remnawave_infra_provider.chain.uuid
  node_uuid       = remnawave_node.chain.uuid
  next_billing_at = "2026-09-01T00:00:00.000Z"
}
resource "remnawave_billing_history" "chain" {
  provider_uuid = remnawave_infra_provider.chain.uuid
  amount        = 99.99
  billed_at     = "2026-07-15T00:00:00.000Z"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_infra_provider.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_node.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_node.chain", "provider_uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_node.chain", "node_uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_history.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_history.chain", "provider_uuid"),
			),
		}},
	})
}

// TestAccMetadataChain verifies metadata resources attached across two entity types:
// node → node_metadata, and user → user_metadata.
func TestAccMetadataChain(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("metadata-chain-profile", "VLESS_TCP_META_CHAIN_ACC") + `
resource "remnawave_node" "chain" {
  name                    = "metadata-chain-node"
  address                 = "127.0.0.20"
  port                    = 2224
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
resource "remnawave_node_metadata" "chain" {
  node_uuid = remnawave_node.chain.uuid
  metadata  = jsonencode({ environment = "production", region = "eu-west" })
}
resource "remnawave_user" "chain" {
  username            = "metadata-chain-user"
  expire_at           = "2027-06-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
resource "remnawave_user_metadata" "chain" {
  user_uuid = remnawave_user.chain.uuid
  metadata  = jsonencode({ department = "platform", tier = "premium" })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_node.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node_metadata.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node_metadata.chain", "node_uuid"),
				resource.TestCheckResourceAttrSet("remnawave_user.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_user_metadata.chain", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_user_metadata.chain", "user_uuid"),
			),
		}},
	})
}
