package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExternalSquadResource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test" {
  name = "test-external-squad"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_external_squad.test", "name", "test-external-squad"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test", "uuid"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test" {
  name = "renamed-external-squad"
}
`,
				Check: resource.TestCheckResourceAttr("remnawave_external_squad.test", "name", "renamed-external-squad"),
			},
		},
	})
}
