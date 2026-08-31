package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostInternalSquads exercises the Remnawave 3.4 host
// internalSquads {mode, squads} contract: the dedicated block in both modes,
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
					resource.TestCheckResourceAttrSet("remnawave_host.test", "internal_squads.0"),
					resource.TestCheckResourceAttr("remnawave_host.test", "excluded_internal_squads.#", "1"),
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
