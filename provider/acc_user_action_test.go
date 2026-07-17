package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserActionResource_ResetTraffic verifies the reset_traffic action
// on a freshly created user.
func TestAccUserActionResource_ResetTraffic(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "action-reset-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user_action" "reset" {
  user_uuid = remnawave_user.test.uuid
  action    = "reset_traffic"
  triggers  = ["initial"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.reset", "action", "reset_traffic"),
					resource.TestCheckResourceAttr("remnawave_user_action.reset", "triggers.0", "initial"),
					resource.TestCheckResourceAttrSet("remnawave_user_action.reset", "id"),
					resource.TestCheckResourceAttrSet("remnawave_user_action.reset", "user_uuid"),
				),
			},
		},
	})
}

// TestAccUserActionResource_RevokeSubscription verifies the revoke_subscription action.
func TestAccUserActionResource_RevokeSubscription(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "action-revoke-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user_action" "revoke" {
  user_uuid = remnawave_user.test.uuid
  action    = "revoke_subscription"
  triggers  = ["initial"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.revoke", "action", "revoke_subscription"),
					resource.TestCheckResourceAttrSet("remnawave_user_action.revoke", "id"),
				),
			},
		},
	})
}

// TestAccUserActionResource_DisableEnable verifies disable then enable actions
// across two resources targeting the same user.
func TestAccUserActionResource_DisableEnable(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "action-disable-enable-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user_action" "disable" {
  user_uuid = remnawave_user.test.uuid
  action    = "disable"
  triggers  = ["initial"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.disable", "action", "disable"),
					resource.TestCheckResourceAttrSet("remnawave_user_action.disable", "id"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "action-disable-enable-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user_action" "enable" {
  user_uuid = remnawave_user.test.uuid
  action    = "enable"
  triggers  = ["initial"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user_action.enable", "action", "enable"),
					resource.TestCheckResourceAttrSet("remnawave_user_action.enable", "id"),
				),
			},
		},
	})
}
