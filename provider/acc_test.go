package provider

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func isBackendAtLeast3() bool {
	return isBackendAtLeast(3, 0)
}

func isBackendAtLeast3_1() bool {
	return isBackendAtLeast(3, 1)
}

func isBackendAtLeast3_2_2() bool {
	version := os.Getenv("REMNAWAVE_VERSION")
	if version == "" {
		return true // docker-compose.yaml defaults to Remnawave 3.2.3
	}
	major, minor, patch, ok := parseVersion(version)
	if !ok {
		return false
	}
	return major > 3 || major == 3 && (minor > 2 || minor == 2 && patch >= 2)
}

func isBackendAtLeast3_2_3() bool {
	version := os.Getenv("REMNAWAVE_VERSION")
	if version == "" {
		return true // docker-compose.yaml defaults to Remnawave 3.2.3
	}
	major, minor, patch, ok := parseVersion(version)
	if !ok {
		return false
	}
	return major > 3 || major == 3 && (minor > 2 || minor == 2 && patch >= 3)
}

func isBackendAtLeast(requiredMajor, requiredMinor int) bool {
	version := strings.TrimPrefix(os.Getenv("REMNAWAVE_VERSION"), "v")
	if version == "" {
		return true // docker-compose.yaml defaults to Remnawave 3.2.3
	}
	majorPart, remainder, ok := strings.Cut(version, ".")
	if !ok {
		return false
	}
	minorPart, _, _ := strings.Cut(remainder, ".")
	major, majorErr := strconv.Atoi(majorPart)
	minor, minorErr := strconv.Atoi(minorPart)
	if majorErr != nil || minorErr != nil {
		return false
	}
	return major > requiredMajor || major == requiredMajor && minor >= requiredMinor
}

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
			password = "TestAdminPassword1234567"
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
