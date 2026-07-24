package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccInternalSquadsDataSource verifies the internal_squads data source lists
// squads after creating a fixture squad, including the derived accessible node list.
func TestAccInternalSquadsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_internal_squad" "test" {
  name     = "ds-test-internal-squad"
  inbounds = []
}

data "remnawave_internal_squads" "all" {
  depends_on = [remnawave_internal_squad.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_internal_squads.all", "internal_squads.#"),
					resource.TestCheckResourceAttrSet("data.remnawave_internal_squads.all", "internal_squads.0.uuid"),
					// accessible_node_uuids depends on the stand (nodes/inbounds), so
					// only assert the attribute is present, not its contents.
					resource.TestCheckResourceAttrSet("data.remnawave_internal_squads.all", "internal_squads.0.accessible_node_uuids.#"),
				),
			},
		},
	})
}

// TestAccExternalSquadsDataSource verifies the external_squads data source lists
// squads after creating a fixture squad.
func TestAccExternalSquadsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test" {
  name = "ds-test-external-squad"
}

data "remnawave_external_squads" "all" {
  depends_on = [remnawave_external_squad.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_external_squads.all", "external_squads.#"),
					resource.TestCheckResourceAttrSet("data.remnawave_external_squads.all", "external_squads.0.uuid"),
				),
			},
		},
	})
}
