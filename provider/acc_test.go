package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// testAccProviderConfig returns the provider configuration block for
	// acceptance tests. Credentials are injected via environment variables
	// — same pattern as the 3x-ui provider.
	testAccProviderConfig = `
provider "remnawave" {
  endpoint = "%s"
  %s
}
`
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"remnawave": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	if os.Getenv(envEndpoint) == "" {
		t.Skipf("%s not set", envEndpoint)
	}
}

func testAccProviderBlock() (string, string) {
	endpoint := os.Getenv(envEndpoint)
	if endpoint == "" {
		endpoint = "http://localhost:3000"
	}

	// Build auth block: api_token takes priority, then username/password
	authBlock := ""
	if token := os.Getenv(envAPIToken); token != "" {
		authBlock = "api_token = \"" + token + "\""
	} else {
		username := os.Getenv(envUsername)
		if username == "" {
			username = "admin"
		}
		password := os.Getenv(envPassword)
		if password == "" {
			password = "TestAdminPassword123456"
		}
		authBlock = "username = \"" + username + "\"\n  password = \"" + password + "\""
	}

	insecure := os.Getenv(envInsecureSkipVerify)
	if insecure == "true" {
		authBlock += "\n  insecure_skip_verify = true"
	}

	// For acceptance tests against a local panel without reverse proxy
	if os.Getenv("REMNAWAVE_TEST_PROXY_HEADERS") == "true" {
		authBlock += "\n  # Proxy headers for test environment"
	}

	return endpoint, authBlock
}
