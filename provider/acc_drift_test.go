package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccBackendClient builds a provider Client from the acceptance-test
// environment, authenticated the same way as the provider under test. It is
// used to mutate the Remnawave backend out-of-band so drift tests can verify
// that the provider's Read path reconciles external changes.
func testAccBackendClient(t *testing.T) *Client {
	t.Helper()

	endpoint := os.Getenv(envEndpoint)
	if endpoint == "" {
		endpoint = "http://localhost:3000"
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
		Endpoint:           endpoint,
		APIToken:           os.Getenv(envAPIToken),
		Username:           username,
		Password:           password,
		InsecureSkipVerify: os.Getenv(envInsecureSkipVerify) == "true",
		ProxyHeaders:       os.Getenv(envProxyHeaders) == "true",
	})
	if err != nil {
		t.Fatalf("create backend mutation client: %v", err)
	}
	return client
}

// testAccResourceUUID returns a CheckFunc that captures the given resource's
// "uuid" attribute into the provided pointer, so a later TestStep's PreConfig
// can mutate that exact backend object.
func testAccResourceUUID(resourceName string, dest *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instance, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		uuid := instance.Primary.Attributes["uuid"]
		if uuid == "" {
			return fmt.Errorf("resource %s has no uuid", resourceName)
		}
		*dest = uuid
		return nil
	}
}

// TestAccUserResource_DriftRefresh verifies that the provider reconciles an
// out-of-band change to a user's description: after mutating the backend
// directly, a terraform refresh must surface the drifted value in state.
// This exercises the remnawave_user Read path against drift.
func TestAccUserResource_DriftRefresh(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const cfg = `
resource "remnawave_user" "test" {
  username            = "drift-user-acc"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
  description         = "drift-orig"
  tag                 = "DRIFT_TAG"
}
`

	var userUUID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_user.test", "description", "drift-orig"),
					resource.TestCheckResourceAttr("remnawave_user.test", "tag", "DRIFT_TAG"),
					testAccResourceUUID("remnawave_user.test", &userUUID),
				),
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := testAccBackendClient(t)
					user, err := client.GetUserByUUID(ctx, userUUID)
					if err != nil {
						t.Fatalf("fetch user for drift mutation: %v", err)
					}
					drifted := "drifted-via-backend"
					user.Description = &drifted
					// Mirror resource_user.go Update: these fields are not accepted by
					// UpdateUserCommand and must be absent from the PATCH payload, or
					// the backend treats them as changes and the resource drifts into
					// RequiresReplace (short_uuid/last_traffic_reset_at).
					user.CreatedAt = ""
					user.LastTrafficResetAt = nil
					user.TrojanPassword = ""
					user.VlessUUID = ""
					user.SsPassword = ""
					if _, err := client.UpdateUser(ctx, user); err != nil {
						t.Fatalf("mutate user description out-of-band: %v", err)
					}
				},
				RefreshState: true,
				// After an out-of-band mutation, the refreshed state diverges from
				// the HCL config, so the post-refresh plan is intentionally
				// non-empty. The non-empty plan is itself proof that Read saw the
				// drift; the Check then asserts the drifted value landed in state.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					// After refresh, state must reflect the backend-mutated value,
					// proving the Read path reconciles drift.
					resource.TestCheckResourceAttr("remnawave_user.test", "description", "drifted-via-backend"),
				),
			},
		},
	})
}

// TestAccNodeResource_DriftRefresh verifies that the provider reconciles an
// out-of-band change to a node's name via a terraform refresh.
func TestAccNodeResource_DriftRefresh(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const nodeCfg = `
resource "remnawave_node" "test" {
  name                       = "drift-node-orig"
  address                    = "127.0.0.30"
  port                       = 2230
  country_code               = "NL"
  is_traffic_tracking_active = true
  traffic_limit_bytes        = 1073741824
  traffic_reset_day          = 15
  notify_percent             = 80
  consumption_multiplier     = 1.0
  config_profile_uuid        = remnawave_config_profile.profile.uuid
  config_profile_inbounds    = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`

	var nodeUUID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("drift-node-profile", "VLESS_TCP_DRIFT_NODE") + nodeCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node.test", "name", "drift-node-orig"),
					testAccResourceUUID("remnawave_node.test", &nodeUUID),
				),
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := testAccBackendClient(t)
					node, err := client.GetNodeByUUID(ctx, nodeUUID)
					if err != nil {
						t.Fatalf("fetch node for drift mutation: %v", err)
					}
					node.Name = "drift-node-mutated"
					consumptionMultiplier := 2.5
					node.ConsumptionMultiplier = &consumptionMultiplier
					if _, err := client.UpdateNode(ctx, node); err != nil {
						t.Fatalf("mutate node out-of-band: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node.test", "name", "drift-node-mutated"),
					resource.TestCheckResourceAttr("remnawave_node.test", "consumption_multiplier", "2.5"),
				),
			},
		},
	})
}

// TestAccHostResource_DriftRefresh verifies that the provider reconciles
// out-of-band changes to a host's remark and is_hidden flags via refresh.
func TestAccHostResource_DriftRefresh(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const hostCfg = `
resource "remnawave_host" "test" {
  remark                        = "drift-host-orig"
  address                       = "drift-host.example.com"
  port                          = 443
  sni                           = "drift-host.example.com"
  security_layer                = "TLS"
  override_sni_from_address     = true
  keep_sni_blank                = false
  vless_route_id                = 7
%s
  xray_json_template_uuid       = remnawave_subscription_template.host.uuid
  exclude_from_subscription_types = ["MIHOMO", "SINGBOX"]
  config_profile_uuid           = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid   = remnawave_config_profile.profile.inbounds[0].uuid
}

resource "remnawave_subscription_template" "host" {
  name          = "drift-host-template"
  template_type = "XRAY_JSON"
}
`

	var hostUUID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("drift-host-profile", "VLESS_TCP_DRIFT_HOST") +
					fmt.Sprintf(hostCfg, hostV28Fields("auto", "true", "true", "false")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host.test", "remark", "drift-host-orig"),
					resource.TestCheckResourceAttr("remnawave_host.test", "is_hidden", "false"),
					testAccResourceUUID("remnawave_host.test", &hostUUID),
				),
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := testAccBackendClient(t)
					host, err := client.GetHostByUUID(ctx, hostUUID)
					if err != nil {
						t.Fatalf("fetch host for drift mutation: %v", err)
					}
					host.Remark = "drift-host-mutated"
					host.IsHidden = true
					if _, err := client.UpdateHost(ctx, host); err != nil {
						t.Fatalf("mutate host remark/is_hidden out-of-band: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host.test", "remark", "drift-host-mutated"),
					resource.TestCheckResourceAttr("remnawave_host.test", "is_hidden", "true"),
				),
			},
		},
	})
}

// TestAccConfigProfile_DriftRefresh verifies that the provider reconciles an
// out-of-band change to a config profile's name via refresh.
func TestAccConfigProfile_DriftRefresh(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const profileCfg = `
resource "remnawave_config_profile" "test" {
  name = "drift-profile-orig"
  config = jsonencode({
    log      = { loglevel = "warning" }
    inbounds = [{
      tag      = "VLESS_TCP_DRIFT_PROFILE"
      listen   = "0.0.0.0"
      port     = 443
      protocol = "vless"
      settings = { decryption = "none", clients = [] }
      streamSettings = { network = "tcp", security = "none" }
    }]
  })
}
`

	var profileUUID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + profileCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_config_profile.test", "name", "drift-profile-orig"),
					testAccResourceUUID("remnawave_config_profile.test", &profileUUID),
				),
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := testAccBackendClient(t)
					profile, err := client.GetConfigProfileByUUID(ctx, profileUUID)
					if err != nil {
						t.Fatalf("fetch config profile for drift mutation: %v", err)
					}
					profile.Name = "drift-profile-mutated"
					if _, err := client.UpdateConfigProfile(ctx, profile); err != nil {
						t.Fatalf("mutate config profile name out-of-band: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_config_profile.test", "name", "drift-profile-mutated"),
				),
			},
		},
	})
}

// TestAccSubscriptionSettings_DriftRefresh verifies that the provider
// reconciles an out-of-band change to the singleton subscription settings'
// profile_title via refresh.
func TestAccSubscriptionSettings_DriftRefresh(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	const settingsCfg = `
resource "remnawave_subscription_settings" "test" {
  profile_title = "drift-settings-orig"
  support_link  = "https://t.me/drift"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + settingsCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "profile_title", "drift-settings-orig"),
				),
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := testAccBackendClient(t)
					current, err := client.GetSubscriptionSettings(ctx)
					if err != nil {
						t.Fatalf("fetch subscription settings for drift mutation: %v", err)
					}
					// Build a minimal PATCH payload (mirroring
					// resource_subscription_settings Update): only the fields we
					// intend to change, plus the singleton UUID. Re-serializing the
					// full fetched object (with its json.RawMessage fields) is
					// rejected by the backend with 400.
					drifted := "drift-settings-mutated"
					support := "https://t.me/drift"
					if _, err := client.UpdateSubscriptionSettings(ctx, &SubscriptionSettings{
						UUID:         current.UUID,
						ProfileTitle: &drifted,
						SupportLink:  &support,
					}); err != nil {
						t.Fatalf("mutate subscription settings profile_title out-of-band: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subscription_settings.test", "profile_title", "drift-settings-mutated"),
				),
			},
		},
	})
}
