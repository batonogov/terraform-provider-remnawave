package provider

import (
	"fmt"
	"regexp"
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
  user_uuid = remnawave_user.hwid.uuid
  hwid      = "terraform-acceptance-device"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_hwid_device.test", "id"),
				resource.TestCheckResourceAttr("remnawave_hwid_device.test", "hwid", "terraform-acceptance-device"),
				resource.TestCheckResourceAttrSet("remnawave_hwid_device.test", "user_uuid"),
			),
		},
			{
				ResourceName:      "remnawave_hwid_device.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceAttrImportStateID("remnawave_hwid_device.test", "id"),
			},
			{
				// Metadata fields are Computed-only (panel-owned; no Update endpoint).
				// Setting any of them in config must be rejected at validation time.
				Config: providerCfg + `
resource "remnawave_user" "hwid" {
  username          = "hwid-acc-user"
  expire_at         = "2028-01-01T00:00:00.000Z"
  hwid_device_limit = 2
}

resource "remnawave_hwid_device" "test" {
  user_uuid  = remnawave_user.hwid.uuid
  hwid       = "terraform-acceptance-device"
  user_agent = "must-not-be-settable"
}
`,
				ExpectError: regexp.MustCompile(`(?i)value for unconfigurable|cannot be set|can't configure|calculated by the provider`),
			},
		},
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
