package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHwidDeviceResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "hwid" {
  username          = "hwid-acc-user"
  expire_at         = "2028-01-01T00:00:00.000Z"
  hwid_device_limit = 2
}

resource "remnawave_hwid_device" "test" {
  user_uuid    = remnawave_user.hwid.uuid
  hwid         = "terraform-acceptance-device"
  platform     = "linux"
  os_version   = "test"
  device_model = "terraform"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_hwid_device.test", "id"),
				resource.TestCheckResourceAttr("remnawave_hwid_device.test", "hwid", "terraform-acceptance-device"),
				resource.TestCheckResourceAttr("remnawave_hwid_device.test", "platform", "linux"),
			),
		}},
	})
}

func TestAccHwidStatsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps:                    []resource.TestStep{{Config: providerCfg + `data "remnawave_hwid_stats" "test" {}`}},
	})
}

func TestAccHwidTopUsersDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps:                    []resource.TestStep{{Config: providerCfg + `data "remnawave_hwid_top_users" "test" {}`}},
	})
}
