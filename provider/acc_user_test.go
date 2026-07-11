package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserResource tests the full lifecycle of a remnawave_user resource:
// create → read → update → import → destroy.
func TestAccUserResource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username       = "testuser-acc"
  expire_at      = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
  description    = "Acceptance test user"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user.test", "username", "testuser-acc"),
					resource.TestCheckResourceAttr("remnawave_user.test", "traffic_limit_bytes", "10737418240"),
					resource.TestCheckResourceAttr("remnawave_user.test", "description", "Acceptance test user"),
					resource.TestCheckResourceAttrSet("remnawave_user.test", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_user.test", "short_uuid"),
					resource.TestCheckResourceAttrSet("remnawave_user.test", "subscription_url"),
				),
			},
			// Import test disabled — needs explicit ID mapping, will add in follow-up
			{
				ResourceName:      "remnawave_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"trojan_password", "vless_uuid", "ss_password"},
			},
		},
	})
}

// TestAccUserResource_Update tests updating a user's fields.
func TestAccUserResource_Update(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username       = "testuser-update"
  expire_at      = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 0
}
`,
				Check: resource.TestCheckResourceAttr("remnawave_user.test", "traffic_limit_bytes", "0"),
			},
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username       = "testuser-update"
  expire_at      = "2028-06-15T00:00:00.000Z"
  traffic_limit_bytes = 5368709120
  description    = "Updated description"
  tag            = "ACC_TEST"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user.test", "expire_at", "2028-06-15T00:00:00.000Z"),
					resource.TestCheckResourceAttr("remnawave_user.test", "traffic_limit_bytes", "5368709120"),
					resource.TestCheckResourceAttr("remnawave_user.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("remnawave_user.test", "tag", "ACC_TEST"),
				),
			},
		},
	})
}
