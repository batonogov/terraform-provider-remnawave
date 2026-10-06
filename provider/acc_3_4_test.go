package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccHostInternalSquads exercises the Remnawave 3.4 host
// internalSquads {mode, squads} contract: the flat attributes in both modes,
// the deprecated excluded_internal_squads translation, and the mirror view
// the provider keeps for pre-3.4 configurations.
func TestAccHostInternalSquads(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_4() {
		t.Skip("host internal_squads require Remnawave 3.4+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	fixtures := providerCfg + testAccProfileConfig("host-squads-profile", "VLESS_TCP_HOST_SQUADS_ACC") + `
resource "remnawave_internal_squad" "test" {
  name     = "test-int-squad-3-4"
  inbounds = []
}
`

	baseHost := `
resource "remnawave_host" "test" {
  remark                      = "terraform-host-squads"
  address                     = "host.example.com"
  port                        = 443
  sni                         = "host.example.com"
  security_layer              = "TLS"
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
%s
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fixtures + fmt.Sprintf(baseHost, `
  internal_squads_mode = "EXCLUDE"
  internal_squads      = [remnawave_internal_squad.test.uuid]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_host.test", "uuid"),
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads_mode", "EXCLUDE"),
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads.#", "1"),
					resource.TestCheckResourceAttrPair("remnawave_host.test", "internal_squads.0", "remnawave_internal_squad.test", "uuid"),
					resource.TestCheckResourceAttr("remnawave_host.test", "excluded_internal_squads.#", "1"),
					resource.TestCheckResourceAttrPair("remnawave_host.test", "excluded_internal_squads.0", "remnawave_internal_squad.test", "uuid"),
				),
			},
			{
				Config: fixtures + fmt.Sprintf(baseHost, `
  internal_squads_mode = "ALLOW_ONLY"
  internal_squads      = [remnawave_internal_squad.test.uuid]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads_mode", "ALLOW_ONLY"),
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads.#", "1"),
					resource.TestCheckResourceAttr("remnawave_host.test", "excluded_internal_squads.#", "0"),
				),
			},
			{
				// Deprecated attribute only: the provider must translate it to
				// mode = "EXCLUDE" and keep the mirror view.
				Config: fixtures + fmt.Sprintf(baseHost, `
  excluded_internal_squads = [remnawave_internal_squad.test.uuid]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads_mode", "EXCLUDE"),
					resource.TestCheckResourceAttr("remnawave_host.test", "internal_squads.#", "1"),
					resource.TestCheckResourceAttr("remnawave_host.test", "excluded_internal_squads.#", "1"),
				),
			},
			{
				ResourceName:                         "remnawave_host.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_host.test"),
			},
		},
	})
}

// TestAccHostInternalSquads_Pre3_4Rejected pins the version gate on panels
// older than 3.4: the squad attributes must fail with the provider's
// version message, not with an opaque backend error.
func TestAccHostInternalSquads_Pre3_4Rejected(t *testing.T) {
	testAccPreCheck(t)
	if isBackendAtLeast3_4() {
		t.Skip("rejection path only observable on panels older than 3.4")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-squads-reject-profile", "VLESS_TCP_HOST_SQUADS_REJECT") + `
resource "remnawave_host" "test" {
  remark                      = "terraform-host-squads-reject"
  address                     = "host.example.com"
  port                        = 443
  internal_squads             = []
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
}
`,
				ExpectError: regexp.MustCompile(`requires Remnawave 3.4 or later`),
			},
		},
	})
}

// TestAccSharedListSlashedName proves the Remnawave 3.4 shared-list
// identifier contract end to end: a name containing "/" must survive
// create, read, data-source listing, update, import, and the body-based
// delete.
func TestAccSharedListSlashedName(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_4() {
		t.Skip("slashed shared-list names require Remnawave 3.4+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_shared_list" "test" {
  name = "terraform/private_ranges"
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
					resource.TestCheckResourceAttr("remnawave_shared_list.test", "name", "terraform/private_ranges"),
					resource.TestCheckResourceAttrSet("remnawave_shared_list.test", "config"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.0.name", "terraform/private_ranges"),
					resource.TestCheckResourceAttr("data.remnawave_shared_lists.all", "shared_lists.0.items_count", "2"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_shared_list" "test" {
  name = "terraform/private_ranges"
  config = jsonencode({
    type  = "ipList"
    items = ["10.0.0.0/8"]
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_shared_list.test", "name", "terraform/private_ranges"),
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

// TestAccNodePluginPostStart exercises the Remnawave 3.4.5 postStart plugin
// contract: a webhook-only section (the backend stores its schema default
// enabled=false, which the provider must drop to keep the plan stable) and an
// explicit enabled flag.
func TestAccNodePluginPostStart(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_4_5() {
		t.Skip("postStart node plugin configuration requires Remnawave 3.4.5+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_node_plugin" "post_start" {
  name = "test-plugin-post-start"
  plugin_config = jsonencode({
    sharedLists = []
    postStart = {
      webhook = {
        enabled = true
        url     = "https://example.com/core-started"
      }
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.post_start", "uuid"),
					resource.TestCheckResourceAttr("remnawave_node_plugin.post_start", "name", "test-plugin-post-start"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.post_start", "plugin_config"),
					testAccCheckNodePluginConfigOmitsPostStartEnabled("remnawave_node_plugin.post_start"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_node_plugin" "post_start" {
  name = "test-plugin-post-start"
  plugin_config = jsonencode({
    sharedLists = []
    postStart = {
      enabled = true
      webhook = {
        enabled = true
        url     = "https://example.com/core-started-v2"
      }
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.post_start", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.post_start", "plugin_config"),
				),
			},
		},
	})
}

// testAccCheckNodePluginConfigOmitsPostStartEnabled proves the state does not
// keep the postStart.enabled default Remnawave 3.4.5 materializes when the
// configuration omits it. In canonical JSON enabled would sort before
// webhook, so the presence of "postStart":{"enabled" is decisive.
func testAccCheckNodePluginConfigOmitsPostStartEnabled(name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found in state", name)
		}
		if strings.Contains(rs.Primary.Attributes["plugin_config"], `"postStart":{"enabled"`) {
			return fmt.Errorf("plugin_config kept a backend-injected postStart.enabled: %s", rs.Primary.Attributes["plugin_config"])
		}
		return nil
	}
}
