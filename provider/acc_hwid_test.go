package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHwidDeviceResource tests creating a HWID device for a user.
// It creates a user first, then a device, then destroys both.
func TestAccHwidDeviceResource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "hwid-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_hwid_device" "test" {
  user_uuid = remnawave_user.test.uuid
  hwid      = "test-hwid-12345678"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_hwid_device.test", "hwid", "test-hwid-12345678"),
					resource.TestCheckResourceAttrSet("remnawave_hwid_device.test", "user_uuid"),
					resource.TestCheckResourceAttrSet("remnawave_hwid_device.test", "id"),
				),
			},
		},
	})
}

// TestAccHwidStatsDataSource tests the hwid_stats data source.
func TestAccHwidStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_hwid_stats" "current" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_hwid_stats.current", "response"),
				),
			},
		},
	})
}

// TestAccHwidTopUsersDataSource tests the hwid_top_users data source.
func TestAccHwidTopUsersDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_hwid_top_users" "current" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_hwid_top_users.current", "response"),
				),
			},
		},
	})
}
