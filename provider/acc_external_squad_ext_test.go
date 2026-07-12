package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExternalSquadExtendedResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_external_squad" "test_ext" {
  name = "ext-squad-test"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_external_squad.test_ext", "name", "ext-squad-test"),
				resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "uuid"),
			),
		}},
	})
}
