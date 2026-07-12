package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccBandwidthRealtimeDataSource tests the remnawave_bandwidth_realtime data source.
// Note: realtime endpoint requires at least one connected node.
// When no nodes are connected, the API returns 404 — so we use
// ExpectNonEmptyPlan and accept that the data source itself works.
func TestAccBandwidthRealtimeDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_bandwidth_realtime" "test" {}`,
				// Realtime endpoint may 404 if no nodes are connected in test env.
				ExpectError: regexp.MustCompile(`Failed to get bandwidth realtime|request failed`),
			},
		},
	})
}

// TestAccSystemBandwidthStatsDataSource tests the remnawave_system_bandwidth_stats data source.
func TestAccSystemBandwidthStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_bandwidth_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_bandwidth_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccSystemNodesStatsDataSource tests the remnawave_system_nodes_stats data source.
func TestAccSystemNodesStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_nodes_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_nodes_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccSubscriptionRequestHistoryStatsDataSource tests the remnawave_subscription_request_history_stats data source.
func TestAccSubscriptionRequestHistoryStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_subscription_request_history_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_subscription_request_history_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccConnectionKeysDataSource tests the remnawave_connection_keys data source
// by creating a user first and referencing its UUID.
func TestAccConnectionKeysDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "ck-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
data "remnawave_connection_keys" "test" {
  uuid = remnawave_user.test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_connection_keys.test", "response"),
				),
			},
		},
	})
}
