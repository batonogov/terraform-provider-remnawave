package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserMetadataResource tests create → read → update cycle for user metadata.
func TestAccUserMetadataResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "meta-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
resource "remnawave_user_metadata" "test" {
  user_uuid = remnawave_user.test.uuid
  metadata  = jsonencode({ department = "engineering" })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_user_metadata.test", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_user_metadata.test", "metadata"),
				),
			},
			{
				ResourceName:            "remnawave_user_metadata.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceAttrImportStateID("remnawave_user_metadata.test", "user_uuid"),
			},
		},
	})
}

func TestAccNodeMetadataResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("metadata-profile", "VLESS_TCP_META_ACC") + `
resource "remnawave_node" "metadata" {
  name                    = "metadata-node"
  address                 = "127.0.0.11"
  port                    = 2223
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

resource "remnawave_node_metadata" "test" {
  node_uuid = remnawave_node.metadata.uuid
  metadata  = jsonencode({ environment = "acceptance" })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_node_metadata.test", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node_metadata.test", "metadata"),
				),
			},
			{
				ResourceName:            "remnawave_node_metadata.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceAttrImportStateID("remnawave_node_metadata.test", "node_uuid"),
			},
		},
	})
}
