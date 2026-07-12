package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubpageConfigResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_subpage_config" "test" {
  name   = "test-subpage"
  config = jsonencode({ title = "Test" })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_subpage_config.test", "name", "test-subpage"),
				resource.TestCheckResourceAttrSet("remnawave_subpage_config.test", "uuid"),
			),
		}},
	})
}
