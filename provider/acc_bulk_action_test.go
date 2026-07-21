package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccUserBulkActionResource_ExtendExpiration verifies the action mutates
// the backend using its version-independent wire contract.
func TestAccUserBulkActionResource_ExtendExpiration(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "bulk" {
  username            = "bulk-extend-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user_bulk_action" "extend" {
  action   = "extend_expiration"
  uuids    = [remnawave_user.bulk.uuid]
  days     = 7
  triggers = { init = "1" }
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_user_bulk_action.extend", "action", "extend_expiration"),
				resource.TestCheckResourceAttr("remnawave_user_bulk_action.extend", "uuids.#", "1"),
				resource.TestCheckResourceAttr("remnawave_user_bulk_action.extend", "days", "7"),
				resource.TestCheckResourceAttrSet("remnawave_user_bulk_action.extend", "id"),
				testAccCheckUserExpiration("remnawave_user.bulk", "2027-01-08T00:00:00Z"),
			),
			// The imperative action intentionally changes the managed user's
			// expiration outside its declarative configuration.
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccCheckUserExpiration(resourceName, want string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instance, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		uuid := instance.Primary.Attributes["uuid"]
		if uuid == "" {
			return fmt.Errorf("resource %s has no uuid", resourceName)
		}

		username := os.Getenv(envUsername)
		if username == "" {
			username = "admin"
		}
		password := os.Getenv(envPassword)
		if password == "" {
			password = "TestAdminPassword1234567"
		}
		client, err := NewClient(ClientConfig{
			Endpoint:           os.Getenv(envEndpoint),
			APIToken:           os.Getenv(envAPIToken),
			Username:           username,
			Password:           password,
			InsecureSkipVerify: os.Getenv(envInsecureSkipVerify) == "true",
			ProxyHeaders:       os.Getenv(envProxyHeaders) == "true",
		})
		if err != nil {
			return fmt.Errorf("create acceptance client: %w", err)
		}
		user, err := client.GetUserByUUID(context.Background(), uuid)
		if err != nil {
			return fmt.Errorf("read user after bulk extension: %w", err)
		}
		gotTime, err := time.Parse(time.RFC3339Nano, user.ExpireAt)
		if err != nil {
			return fmt.Errorf("parse backend expire_at %q: %w", user.ExpireAt, err)
		}
		wantTime, err := time.Parse(time.RFC3339Nano, want)
		if err != nil {
			return fmt.Errorf("parse expected expire_at %q: %w", want, err)
		}
		if !gotTime.Equal(wantTime) {
			return fmt.Errorf("backend expire_at = %s, want %s", gotTime, wantTime)
		}
		return nil
	}
}

// TestAccNodeBulkActionResource_Enable verifies an idempotent node bulk action
// against a freshly created, enabled node.
func TestAccNodeBulkActionResource_Enable(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("node-bulk-enable", "VLESS_NODE_BULK_ENABLE") + `
resource "remnawave_node" "bulk" {
  name                    = "bulk-enable-node"
  address                 = "127.0.0.24"
  port                    = 2224
  config_profile_uuid     = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}

resource "remnawave_node_bulk_action" "enable" {
  action   = "enable"
  uuids    = [remnawave_node.bulk.uuid]
  triggers = { init = "1" }
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_node_bulk_action.enable", "action", "enable"),
				resource.TestCheckResourceAttr("remnawave_node_bulk_action.enable", "uuids.#", "1"),
				resource.TestCheckResourceAttrSet("remnawave_node_bulk_action.enable", "id"),
			),
		}},
	})
}
