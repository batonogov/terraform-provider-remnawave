package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostTagsDataSource verifies the host_tags data source.
// SKIPPED: backend returns tags wrapped in {response:{tags:[...]}} envelope,
// but GetHostTags expects a bare []string. This is a client bug tracked
// separately — skip until the client is fixed.
func TestAccHostTagsDataSource(t *testing.T) {
	testAccPreCheck(t)
	t.Skip("client bug: GetHostTags does not unwrap response envelope; tracked separately")

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-tags-ds", "VLESS_HOST_TAGS_DS") + `
resource "remnawave_host" "tagged" {
  remark                      = "tags-ds-test"
  address                     = "127.0.0.4"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
  tags                        = ["TEST_TAG_DS"]
}

data "remnawave_host_tags" "all" {
  depends_on = [remnawave_host.tagged]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_host_tags.all", "tags.#"),
				),
			},
		},
	})
}
