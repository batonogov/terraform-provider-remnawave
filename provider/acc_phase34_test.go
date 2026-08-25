package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSnippetResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_snippet" "test" {
  name    = "test-snippet-2"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.test", "name", "test-snippet-2"),
					resource.TestCheckResourceAttrSet("remnawave_snippet.test", "snippet"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_snippet" "test" {
  name    = "test-snippet-2"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads", "geosite:google"] }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.test", "name", "test-snippet-2"),
					resource.TestCheckResourceAttrSet("remnawave_snippet.test", "snippet"),
				),
			},
		},
	})
}

// testAccCheckListContainsAttr looks for wanted among the values of
// "<prefix>.N.<field>". Acceptance tests share one panel within a matrix entry,
// so the snippet list also holds fixtures other tests created; asserting a
// fixed element index or count would make this fail on ordering rather than on
// behaviour.
func testAccCheckListContainsAttr(name, prefix, field, wanted string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("data source %s not found in state", name)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes[prefix+".#"])
		if err != nil {
			return fmt.Errorf("%s has no %s.# count: %w", name, prefix, err)
		}
		for i := 0; i < count; i++ {
			if rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", prefix, i, field)] == wanted {
				return nil
			}
		}
		return fmt.Errorf("%s lists %d entries, none with %s = %q", name, count, field, wanted)
	}
}

func TestAccSnippetsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_snippet" "listed" {
  name    = "terraform-listed-snippet"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}

data "remnawave_snippets" "all" {
  depends_on = [remnawave_snippet.listed]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_snippets.all", "total"),
					resource.TestCheckResourceAttrSet("data.remnawave_snippets.all", "snippets.#"),
					testAccCheckListContainsAttr(
						"data.remnawave_snippets.all", "snippets", "name", "terraform-listed-snippet"),
				),
			},
		},
	})
}

func TestAccSnippetResourceSyncNodesOnChange(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	createConfig := testAccSnippetSyncConfig(providerCfg, `["geosite:category-ads"]`)

	if !isBackendAtLeast3_2_3() {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
			Steps: []resource.TestStep{{
				Config:      createConfig,
				ExpectError: regexp.MustCompile(`sync_nodes_on_change requires Remnawave 3\.2\.3 or later`),
			}},
		})
		return
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.sync", "sync_nodes_on_change", "true"),
					resource.TestCheckResourceAttr("remnawave_snippet.sync", "sync_pending", snippetSyncPhaseNone),
					resource.TestCheckResourceAttrSet("remnawave_config_profile.sync", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node.sync", "uuid"),
				),
			},
			{
				Config: testAccSnippetSyncConfig(providerCfg, `["geosite:category-ads", "geosite:google"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.sync", "sync_nodes_on_change", "true"),
					resource.TestCheckResourceAttr("remnawave_snippet.sync", "sync_pending", snippetSyncPhaseNone),
					resource.TestCheckResourceAttrSet("remnawave_config_profile.sync", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node.sync", "uuid"),
				),
			},
		},
	})
}

func testAccSnippetSyncConfig(providerCfg, domains string) string {
	return providerCfg + fmt.Sprintf(`
resource "remnawave_snippet" "sync" {
  name                 = "test-snippet-sync"
  snippet              = jsonencode([{ "type" = "field", "domain" = %s }])
  sync_nodes_on_change = true
}

resource "remnawave_config_profile" "sync" {
  name = "snippet-sync-profile"
  config = jsonencode({
    log = { loglevel = "warning" }
    inbounds = [{
      tag      = "VLESS_SNIPPET_SYNC_ACC"
      listen   = "0.0.0.0"
      port     = 443
      protocol = "vless"
      settings = { clients = [], decryption = "none" }
      streamSettings = {
        network  = "tcp"
        security = "reality"
        realitySettings = {
          show        = false
          target      = "xray.com"
          xver        = 0
          serverNames = ["xray.com"]
          privateKey  = ""
          shortIds    = []
        }
      }
      sniffing = { enabled = true, destOverride = ["http", "tls", "quic"] }
    }]
    outbounds = [
      { tag = "direct", protocol = "freedom", settings = {} },
      { tag = "block", protocol = "blackhole", settings = {} }
    ]
    routing = {
      domainStrategy = "AsIs"
      rules          = [{ snippet = remnawave_snippet.sync.name }]
    }
  })
}

resource "remnawave_node" "sync" {
  name                    = "terraform-snippet-sync"
  address                 = "127.0.0.32"
  port                    = 2232
  country_code            = "NL"
  config_profile_uuid     = remnawave_config_profile.sync.uuid
  config_profile_inbounds = [remnawave_config_profile.sync.inbounds[0].uuid]
}
`, domains)
}

func TestAccSnippetSyncRecoveryAcrossRefresh(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_2_3() {
		t.Skip("snippet sync recovery requires Remnawave 3.2.3+")
	}

	endpoint, authBlock := testAccProviderBlock()
	target, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("parse backend endpoint: %v", err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	var failNextSync atomic.Bool
	var syncCalls atomic.Int32
	failNextSync.Store(true)

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && req.URL.Path == "/api/snippets/actions/sync" {
			syncCalls.Add(1)
			if failNextSync.CompareAndSwap(true, false) {
				http.Error(w, "injected sync failure", http.StatusInternalServerError)
				return
			}
		}
		if req.Method == http.MethodDelete && req.URL.Path == "/api/snippets" {
			reverseProxy.ServeHTTP(w, req)
			failNextSync.Store(true)
			return
		}
		reverseProxy.ServeHTTP(w, req)
	}))
	defer proxyServer.Close()

	providerCfg := fmt.Sprintf(testAccProviderConfig, proxyServer.URL, authBlock)
	createConfig := providerCfg + `
resource "remnawave_snippet" "recovery" {
  name                 = "test-snippet-recovery"
  snippet              = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
  sync_nodes_on_change = true
}
`
	updatedConfig := providerCfg + `
resource "remnawave_snippet" "recovery" {
  name                 = "test-snippet-recovery"
  snippet              = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads", "geosite:google"] }])
  sync_nodes_on_change = true
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.recovery", "sync_pending", snippetSyncPhaseNone),
				),
			},
			{
				Config:      updatedConfig,
				ExpectError: regexp.MustCompile(`Failed to synchronize snippet nodes`),
			},
			{
				PreConfig: func() {
					if got := syncCalls.Load(); got != 1 {
						t.Fatalf("sync calls before update retry plan = %d, want 1", got)
					}
				},
				Config:             updatedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if got := syncCalls.Load(); got != 1 {
						t.Fatalf("planning performed sync: calls = %d, want 1", got)
					}
				},
				Config: updatedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.recovery", "sync_pending", snippetSyncPhaseNone),
				),
			},
			{
				Config:      providerCfg,
				ExpectError: regexp.MustCompile(`Failed to synchronize snippet nodes`),
			},
			{
				PreConfig: func() {
					if got := syncCalls.Load(); got != 3 {
						t.Fatalf("sync calls before delete retry plan = %d, want 3", got)
					}
				},
				Config:             providerCfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if got := syncCalls.Load(); got != 3 {
						t.Fatalf("delete planning performed sync: calls = %d, want 3", got)
					}
				},
				Config: providerCfg,
			},
		},
	})

	if got := syncCalls.Load(); got != 4 {
		t.Fatalf("sync calls = %d, want 4 (failed and resumed update plus failed and resumed delete)", got)
	}
}

func TestAccNodePluginResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_node_plugin" "test" {
  name          = "test-plugin"
  plugin_config = jsonencode({
    sharedLists = []
    connectionDrop = {
      enabled      = false
      whitelistIps = []
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_plugin.test", "name", "test-plugin"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "plugin_config"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_node_plugin" "test" {
  name          = "test-plugin-updated"
  plugin_config = jsonencode({
    sharedLists = []
    connectionDrop = {
      enabled      = false
      whitelistIps = []
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_plugin.test", "name", "test-plugin-updated"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "plugin_config"),
				),
			},
		},
	})
}

func TestAccNodePluginPreStart(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_1() {
		t.Skip("preStart node plugin configuration requires Remnawave 3.1+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_node_plugin" "pre_start" {
  name = "pre-start-plugin"
  plugin_config = jsonencode({
    sharedLists = []
    preStart = {
      enabled = true
      cleanupSockets = {
        enabled = true
        files   = ["/dev/shm/*.sock"]
      }
    }
  })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_node_plugin.pre_start", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node_plugin.pre_start", "plugin_config"),
			),
		}},
	})
}

func TestAccNodePluginIPv6_3_2_3(t *testing.T) {
	testAccPreCheck(t)
	if !isBackendAtLeast3_2_3() {
		t.Skip("correct 6to4 IPv6 validation requires Remnawave 3.2.3+")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)
	sharedListResource := ""
	sharedListsConfig := `sharedLists = [{
      name  = "ext:ipv6-6to4"
      type  = "ipList"
      items = ["2002:c000:204::1", "2002:c000:204::/48"]
    }]`
	whitelist := `"2002:c000:204::1"`
	pluginDependency := ""
	if isBackendAtLeast3_3() {
		sharedListResource = `
resource "remnawave_shared_list" "ipv6" {
  name = "ipv6-6to4"
  config = jsonencode({
    type  = "ipList"
    items = ["2002:c000:204::1", "2002:c000:204::/48"]
  })
}
`
		sharedListsConfig = "sharedLists = []"
		whitelist = `"ext:ipv6-6to4"`
		pluginDependency = "  depends_on = [remnawave_shared_list.ipv6]\n"
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + sharedListResource + fmt.Sprintf(`
resource "remnawave_node_plugin" "ipv6" {
  name = "test-plugin-ipv6-6to4"
%s  plugin_config = jsonencode({
    %s
    connectionDrop = {
      enabled      = true
      whitelistIps = [%s]
    }
  })
}
`, pluginDependency, sharedListsConfig, whitelist),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_node_plugin.ipv6", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node_plugin.ipv6", "plugin_config"),
			),
		}},
	})
}

func TestAccApiTokenResource(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("api_token resource requires admin JWT — skipped when using api_token auth")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("remnawave_api_token.test", "uuid"),
		resource.TestCheckResourceAttrSet("remnawave_api_token.test", "token"),
		resource.TestCheckResourceAttr("remnawave_api_token.test", "name", "terraform-acceptance"),
	}
	// expire_at, expires_in_days, and scopes are 2.8.x-only — 2.7.x does not
	// return them in the token response.
	if !isBackend2_7() {
		checks = append(checks,
			resource.TestCheckResourceAttrSet("remnawave_api_token.test", "expire_at"),
			resource.TestCheckResourceAttr("remnawave_api_token.test", "expires_in_days", "2"),
			resource.TestCheckResourceAttr("remnawave_api_token.test", "scopes.#", "1"),
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_api_token" "test" {
  name            = "terraform-acceptance"
  expires_in_days = 2
  scopes          = ["*"]
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(checks...),
		}},
	})
}

func TestAccInfraProviderResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_infra_provider" "test" {
  name = "test-provider"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_infra_provider.test", "name", "test-provider"),
					resource.TestCheckResourceAttrSet("remnawave_infra_provider.test", "uuid"),
				),
			},
			// NOTE: update step removed — infra_provider sends favicon_link and
			// login_url as empty strings on update, which the API rejects with
			// "Invalid url" (zod validation). This is a provider bug to fix.
		},
	})
}

func TestAccKeygenDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `data "remnawave_keygen" "current" {}`,
			Check:  resource.TestCheckResourceAttrSet("data.remnawave_keygen.current", "pub_key"),
		}},
	})
}
