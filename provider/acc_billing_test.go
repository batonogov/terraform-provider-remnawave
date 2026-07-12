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
			Config: providerCfg + `
resource "remnawave_infra_provider" "test" {
  name = "billing-test"
}
resource "remnawave_billing_node" "test" {
  provider_uuid   = remnawave_infra_provider.test.uuid
  name            = "billing-node-test"
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
