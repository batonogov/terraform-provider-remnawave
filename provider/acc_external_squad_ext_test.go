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
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test_ext" {
  name = "ext-squad-test"

  templates = jsonencode([{
    templateUuid = remnawave_subscription_template.external.uuid
    templateType = "XRAY_JSON"
  }])
  subscription_settings = jsonencode({
    profileTitle   = "External squad profile"
    randomizeHosts = true
  })
  host_overrides   = jsonencode({ serverDescription = "External server", vlessRouteId = 7 })
  response_headers = jsonencode({ "X-External-Squad" = "terraform" })
  hwid_settings = jsonencode({
    enabled              = true
    fallbackDeviceLimit  = 2
    maxDevicesAnnounce   = null
  })
  custom_remarks = jsonencode({
    expiredUsers          = ["expired"]
    limitedUsers          = ["limited"]
    disabledUsers         = ["disabled"]
    emptyHosts            = ["empty"]
    HWIDMaxDevicesExceeded = ["too-many-devices"]
    HWIDNotSupported      = ["not-supported"]
  })
  subpage_config_uuid = remnawave_subpage_config.external.uuid
}

resource "remnawave_subscription_template" "external" {
  name          = "external-squad-template"
  template_type = "XRAY_JSON"
}

resource "remnawave_subpage_config" "external" {
  name = "external-squad-page"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_external_squad.test_ext", "name", "ext-squad-test"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "templates"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "subscription_settings"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "response_headers"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "subpage_config_uuid"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test_ext" {
  name = "ext-squad-test-updated"

  templates = jsonencode([{
    templateUuid = remnawave_subscription_template.external.uuid
    templateType = "XRAY_JSON"
  }])
  subscription_settings = jsonencode({
    profileTitle   = "Updated external squad profile"
    randomizeHosts = false
  })
  host_overrides   = jsonencode({ serverDescription = "Updated server", vlessRouteId = 8 })
  response_headers = jsonencode({ "X-External-Squad" = "terraform-updated" })
  hwid_settings = jsonencode({
    enabled              = false
    fallbackDeviceLimit  = 5
    maxDevicesAnnounce   = null
  })
  custom_remarks = jsonencode({
    expiredUsers          = ["expired"]
    limitedUsers          = ["limited"]
    disabledUsers         = ["disabled"]
    emptyHosts            = ["empty"]
    HWIDMaxDevicesExceeded = ["too-many-devices"]
    HWIDNotSupported      = ["not-supported"]
  })
  subpage_config_uuid = remnawave_subpage_config.external.uuid
}

resource "remnawave_subscription_template" "external" {
  name          = "external-squad-template"
  template_type = "XRAY_JSON"
}

resource "remnawave_subpage_config" "external" {
  name = "external-squad-page"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_external_squad.test_ext", "name", "ext-squad-test-updated"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "subscription_settings"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test_ext", "response_headers"),
				),
			},
		},
	})
}
