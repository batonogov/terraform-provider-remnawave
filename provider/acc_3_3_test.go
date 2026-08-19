package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNodeIntegrationResourceAndDataSource(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_3() {
		t.Skip("node integrations require Remnawave 3.3+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_node_integration" "test" {
  name        = "terraform-integration"
  description = "Acceptance integration"
  config = jsonencode({
    environmentVariables = {
      TERRAFORM_ACCEPTANCE = "initial"
    }
  })
  restart_nodes_on_update = false
}

data "remnawave_node_integrations" "all" {
  depends_on = [remnawave_node_integration.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_node_integration.test", "uuid"),
					resource.TestCheckResourceAttr("remnawave_node_integration.test", "name", "terraform-integration"),
					resource.TestCheckResourceAttr("remnawave_node_integration.test", "description", "Acceptance integration"),
					resource.TestCheckResourceAttrSet("remnawave_node_integration.test", "config"),
					resource.TestCheckResourceAttr("data.remnawave_node_integrations.all", "node_integrations.#", "1"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_node_integration" "test" {
  name        = "terraform-integration-updated"
  description = "Updated acceptance integration"
  config = jsonencode({
    environmentVariables = {
      TERRAFORM_ACCEPTANCE = "updated"
    }
  })
  restart_nodes_on_update = true
}

data "remnawave_node_integrations" "all" {
  depends_on = [remnawave_node_integration.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_integration.test", "name", "terraform-integration-updated"),
					resource.TestCheckResourceAttr("remnawave_node_integration.test", "restart_nodes_on_update", "true"),
					resource.TestCheckResourceAttr("data.remnawave_node_integrations.all", "node_integrations.#", "1"),
				),
			},
			{
				ResourceName:                         "remnawave_node_integration.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"restart_nodes_on_update"},
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_node_integration.test"),
			},
		},
	})
}

func TestAccSharedListResourceAndDataSource(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_3() {
		t.Skip("global shared lists require Remnawave 3.3+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_shared_list" "test" {
  name = "terraform_private_ranges"
  config = jsonencode({
    type  = "ipList"
    items = ["10.0.0.0/8", "2001:db8::/32"]
  })
}

data "remnawave_shared_lists" "all" {
  depends_on = [remnawave_shared_list.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_shared_list.test", "name", "terraform_private_ranges"),
					resource.TestCheckResourceAttrSet("remnawave_shared_list.test", "config"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.#", "1"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.0.name", "terraform_private_ranges"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.0.type", "ipList"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.0.items_count", "2"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_shared_list" "test" {
  name = "terraform_private_ranges"
  config = jsonencode({
    type  = "ipList"
    items = ["10.0.0.0/8"]
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_shared_list.test", "name", "terraform_private_ranges"),
					resource.TestCheckResourceAttrSet("remnawave_shared_list.test", "config"),
				),
			},
			{
				ResourceName:                         "remnawave_shared_list.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateIdFunc:                    resourceAttrImportStateID("remnawave_shared_list.test", "name"),
			},
		},
	})
}
