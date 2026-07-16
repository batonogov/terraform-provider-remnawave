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
		Steps: []resource.TestStep{
			{
				// Create only accepts name — config is validated by a complex
				// schema (SubscriptionPageRawConfigSchema) on the backend side.
				// We test create with just the name here.
				Config: providerCfg + `
resource "remnawave_subpage_config" "test" {
  name = "test-subpage"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subpage_config.test", "name", "test-subpage"),
					resource.TestCheckResourceAttrSet("remnawave_subpage_config.test", "uuid"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_subpage_config" "test" {
  name = "test-subpage-updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subpage_config.test", "name", "test-subpage-updated"),
					resource.TestCheckResourceAttrSet("remnawave_subpage_config.test", "uuid"),
				),
			},
			{
				ResourceName:            "remnawave_subpage_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceUUIDImportStateID("remnawave_subpage_config.test"),
			},
		},
	})
}
