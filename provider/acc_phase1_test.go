package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSubscriptionSettingsResource tests create → read → update cycle.
func TestAccSubscriptionSettingsResource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	importIgnore := []string{"updated_at"}
	if isBackendAtLeast3() {
		// Removed fields are state-only compatibility values on v3 and cannot
		// be reconstructed by import. Header names are returned lowercase by
		// the backend, while normal refresh preserves configured casing.
		importIgnore = append(importIgnore,
			"profile_title",
			"support_link",
			"profile_update_interval",
			"is_profile_webpage_url_enabled",
			"happ_announce",
			"happ_routing",
			"custom_response_headers",
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_subscription_settings" "test" {
  profile_title = "My VPN Service"
  support_link  = "https://t.me/support"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "id", "settings"),
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "profile_title", "My VPN Service"),
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "support_link", "https://t.me/support"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_subscription_settings" "test" {
  profile_title      = "Updated VPN Title"
  support_link       = "https://t.me/new_support"
  randomize_hosts    = true
  profile_update_interval = 30
  custom_response_headers = jsonencode({ "X-Terraform" = "acceptance" })
  hwid_settings = jsonencode({
    enabled             = true
    fallbackDeviceLimit = 2
    maxDevicesAnnounce  = null
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "profile_title", "Updated VPN Title"),
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "support_link", "https://t.me/new_support"),
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "randomize_hosts", "true"),
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "profile_update_interval", "30"),
					resource.TestCheckResourceAttrSet("remnawave_subscription_settings.test", "custom_response_headers"),
					resource.TestCheckResourceAttrSet("remnawave_subscription_settings.test", "hwid_settings"),
				),
			},
			{
				ResourceName:                         "remnawave_subscription_settings.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              importIgnore,
				ImportStateIdFunc:                    staticImportStateID("settings"),
			},
		},
	})
}

// TestAccUsersDataSource tests users data source after creating a user.
func TestAccUsersDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username  = "ds-test-user"
  expire_at = "2027-01-01T00:00:00.000Z"
}

data "remnawave_users" "all" {
  depends_on = [remnawave_user.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remnawave_users.all", "users.#", "1"),
					resource.TestCheckResourceAttrSet("data.remnawave_users.all", "users.0.uuid"),
					resource.TestCheckResourceAttr("data.remnawave_users.all", "users.0.username", "ds-test-user"),
				),
			},
		},
	})
}

// TestAccHostsDataSource tests hosts data source after creating a host.
func TestAccHostsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_hosts" "all" {}
`,
			},
		},
	})
}
