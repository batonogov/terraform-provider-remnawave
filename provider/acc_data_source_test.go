package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNodesDataSource tests the remnawave_nodes data source against a
// managed node so version-specific response fields can be asserted.
func TestAccNodesDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr("data.remnawave_nodes.all", "nodes.#", "1"),
		resource.TestCheckResourceAttrSet("data.remnawave_nodes.all", "nodes.0.uuid"),
		resource.TestCheckResourceAttr("data.remnawave_nodes.all", "nodes.0.name", "terraform-nodes-ds"),
	}
	if isBackendAtLeast3_1() {
		checks = append(checks, resource.TestCheckResourceAttrSet("data.remnawave_nodes.all", "nodes.0.id"))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("nodes-ds-profile", "VLESS_TCP_NODES_DS_ACC") + `
resource "remnawave_node" "test" {
  name                    = "terraform-nodes-ds"
  address                 = "127.0.0.12"
  port                    = 2224
  country_code            = "NL"
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

data "remnawave_nodes" "all" {
  depends_on = [remnawave_node.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

// TestAccSystemHealthDataSource tests the remnawave_system_health data source.
func TestAccSystemHealthDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_system_health" "current" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_health.current", "response"),
				),
			},
		},
	})
}
