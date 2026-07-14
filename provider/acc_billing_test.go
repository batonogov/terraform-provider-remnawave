package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBillingNodeResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("billing-profile", "VLESS_BILLING_ACC") + `
resource "remnawave_infra_provider" "test" {
  name = "billing-test"
}
resource "remnawave_node" "billing" {
  name                = "billing-node"
  address             = "10.20.30.40"
  port                = 5555
  config_profile_uuid = remnawave_config_profile.profile.uuid
}
resource "remnawave_billing_node" "test" {
  provider_uuid   = remnawave_infra_provider.test.uuid
  node_uuid       = remnawave_node.billing.uuid
  next_billing_at = "2026-08-01T00:00:00.000Z"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_billing_node.test", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_billing_node.test", "provider_uuid"),
			),
		}},
	})
}

func TestAccBillingHistoryResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_infra_provider" "test2" {
  name = "billing-hist-test"
}
resource "remnawave_billing_history" "test" {
  provider_uuid = remnawave_infra_provider.test2.uuid
  amount        = 49.99
  billed_at     = "2026-07-01T00:00:00.000Z"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_billing_history.test", "uuid"),
			),
		}},
	})
}
