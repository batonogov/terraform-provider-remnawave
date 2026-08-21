package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

// TestAccNodePluginTorrentBlocker exercises the torrentBlocker plugin contract
// and the 3.3.1 rulePlacement key. On 3.3.1 the backend injects its own
// rulePlacement default, so the first step also covers the provider's
// normalization of that value.
func TestAccNodePluginTorrentBlocker(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_3() {
		t.Skip("the shared-list free torrentBlocker contract requires Remnawave 3.3+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	config := func(rulePlacement string) string {
		return providerCfg + fmt.Sprintf(`
resource "remnawave_node_plugin" "torrent_blocker" {
  name = "test-plugin-torrent-blocker"
  plugin_config = jsonencode({
    sharedLists = []
    torrentBlocker = {
      enabled       = false
      blockDuration = 3600
      ignoreLists   = {}
%s    }
  })
}
`, rulePlacement)
	}

	steps := []resource.TestStep{{
		Config: config(""),
		Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("remnawave_node_plugin.torrent_blocker", "uuid"),
			testAccCheckNodePluginConfigOmitsRulePlacement("remnawave_node_plugin.torrent_blocker"),
		),
	}}
	if isBackendAtLeast3_3_1() {
		steps = append(steps, resource.TestStep{
			Config: config("      rulePlacement = 5\n"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("remnawave_node_plugin.torrent_blocker", "plugin_config",
					regexp.MustCompile(`"rulePlacement":5`)),
			),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps:                    steps,
	})
}

func testAccCheckNodePluginConfigOmitsRulePlacement(name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found in state", name)
		}
		if strings.Contains(rs.Primary.Attributes["plugin_config"], "rulePlacement") {
			return fmt.Errorf("plugin_config kept a backend-injected rulePlacement: %s", rs.Primary.Attributes["plugin_config"])
		}
		return nil
	}
}
