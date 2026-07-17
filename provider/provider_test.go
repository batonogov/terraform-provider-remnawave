package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderMetadataAndSchema(t *testing.T) {
	t.Parallel()

	p := New("1.2.3")()
	var metadata frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &metadata)
	if metadata.TypeName != "remnawave" || metadata.Version != "1.2.3" {
		t.Errorf("metadata = %#v", metadata)
	}

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	if len(schemaResp.Schema.Attributes) != 7 {
		t.Fatalf("provider attributes = %d, want 7", len(schemaResp.Schema.Attributes))
	}

	endpoint, ok := schemaResp.Schema.Attributes["endpoint"].(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("endpoint attribute type = %T", schemaResp.Schema.Attributes["endpoint"])
	}
	if !endpoint.Optional || endpoint.Required {
		t.Errorf("endpoint must be optional so REMNAWAVE_ENDPOINT can be used: %#v", endpoint)
	}

	for _, name := range []string{"api_token", "password"} {
		attribute, ok := schemaResp.Schema.Attributes[name].(providerschema.StringAttribute)
		if !ok || !attribute.Sensitive {
			t.Errorf("%s must be a sensitive string attribute", name)
		}
	}
}

func TestProviderRegistersUniqueResources(t *testing.T) {
	t.Parallel()

	p := New("test")()
	factories := p.Resources(context.Background())
	if len(factories) != 21 {
		t.Fatalf("resources = %d, want 21", len(factories))
	}

	seen := make(map[string]struct{}, len(factories))
	for _, factory := range factories {
		instance := factory()
		var metadata resource.MetadataResponse
		instance.Metadata(context.Background(), resource.MetadataRequest{}, &metadata)
		if !strings.HasPrefix(metadata.TypeName, "remnawave_") {
			t.Errorf("invalid resource type name %q", metadata.TypeName)
		}
		if _, exists := seen[metadata.TypeName]; exists {
			t.Errorf("duplicate resource type %q", metadata.TypeName)
		}
		seen[metadata.TypeName] = struct{}{}

		var schemaResp resource.SchemaResponse
		instance.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s schema diagnostics: %v", metadata.TypeName, schemaResp.Diagnostics)
		}
		if len(schemaResp.Schema.Attributes) == 0 && len(schemaResp.Schema.Blocks) == 0 {
			t.Errorf("%s has an empty schema", metadata.TypeName)
		}
	}
}

func TestProviderRegistersUniqueDataSources(t *testing.T) {
	t.Parallel()

	p := New("test")()
	factories := p.DataSources(context.Background())
	if len(factories) != 22 {
		t.Fatalf("data sources = %d, want 22", len(factories))
	}

	seen := make(map[string]struct{}, len(factories))
	for _, factory := range factories {
		instance := factory()
		var metadata datasource.MetadataResponse
		instance.Metadata(context.Background(), datasource.MetadataRequest{}, &metadata)
		if !strings.HasPrefix(metadata.TypeName, "remnawave_") {
			t.Errorf("invalid data source type name %q", metadata.TypeName)
		}
		if _, exists := seen[metadata.TypeName]; exists {
			t.Errorf("duplicate data source type %q", metadata.TypeName)
		}
		seen[metadata.TypeName] = struct{}{}

		var schemaResp datasource.SchemaResponse
		instance.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s schema diagnostics: %v", metadata.TypeName, schemaResp.Diagnostics)
		}
		if len(schemaResp.Schema.Attributes) == 0 && len(schemaResp.Schema.Blocks) == 0 {
			t.Errorf("%s has an empty schema", metadata.TypeName)
		}
	}
}

func TestEnvStringPrecedence(t *testing.T) {
	t.Setenv("REMNAWAVE_TEST_VALUE", "from-env")

	tests := []struct {
		name     string
		value    types.String
		fallback string
		want     string
	}{
		{name: "configured value", value: types.StringValue("from-config"), fallback: "fallback", want: "from-config"},
		{name: "configured empty overrides environment", value: types.StringValue(""), fallback: "fallback", want: ""},
		{name: "null uses environment", value: types.StringNull(), fallback: "fallback", want: "from-env"},
		{name: "unknown uses environment", value: types.StringUnknown(), fallback: "fallback", want: "from-env"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envString(tt.value, "REMNAWAVE_TEST_VALUE", tt.fallback); got != tt.want {
				t.Errorf("envString() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Setenv("REMNAWAVE_TEST_VALUE", "")
	if got := envString(types.StringNull(), "REMNAWAVE_TEST_VALUE", "fallback"); got != "fallback" {
		t.Errorf("fallback envString() = %q", got)
	}
}

func TestProviderConfigure(t *testing.T) {
	for _, key := range []string{
		envEndpoint,
		envAPIToken,
		envUsername,
		envPassword,
		envInsecureSkipVerify,
		envRequestTimeout,
		envProxyHeaders,
	} {
		t.Setenv(key, "")
	}

	p := New("test")()
	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Run("missing endpoint", func(t *testing.T) {
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, nil),
		}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Missing endpoint")
	})

	t.Run("missing credentials", func(t *testing.T) {
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{"endpoint": "https://panel.example.com"}),
		}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Missing credentials")
	})

	t.Run("environment configuration", func(t *testing.T) {
		t.Setenv(envEndpoint, "https://env.example.com/base")
		t.Setenv(envAPIToken, "env-token")
		t.Setenv(envInsecureSkipVerify, "true")
		t.Setenv(envRequestTimeout, "45s")
		t.Setenv(envProxyHeaders, "true")

		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, nil),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Configure() diagnostics: %v", resp.Diagnostics)
		}
		client, ok := resp.ResourceData.(*Client)
		if !ok || resp.DataSourceData != resp.ResourceData {
			t.Fatalf("configured data = resource:%T datasource:%T", resp.ResourceData, resp.DataSourceData)
		}
		if client.baseURL.String() != "https://env.example.com/base" || client.apiToken != "env-token" {
			t.Errorf("client endpoint/token = %s/%q", client.baseURL, client.apiToken)
		}
		if client.httpClient.Timeout != 45*time.Second || !client.proxyHeaders {
			t.Errorf("client timeout/proxy = %s/%v", client.httpClient.Timeout, client.proxyHeaders)
		}
		transport := client.httpClient.Transport.(*http.Transport)
		if !transport.TLSClientConfig.InsecureSkipVerify {
			t.Errorf("InsecureSkipVerify = false")
		}
	})

	t.Run("configured username and password", func(t *testing.T) {
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":             "http://panel.example.com",
				"username":             "admin",
				"password":             "secret",
				"request_timeout":      "2m",
				"insecure_skip_verify": false,
				"proxy_headers":        false,
			}),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Configure() diagnostics: %v", resp.Diagnostics)
		}
		client := resp.ResourceData.(*Client)
		if client.username != "admin" || client.password != "secret" || client.httpClient.Timeout != 2*time.Minute {
			t.Errorf("client = %#v", client)
		}
	})

	t.Run("invalid insecure env", func(t *testing.T) {
		t.Setenv(envEndpoint, "https://panel.example.com")
		t.Setenv(envAPIToken, "token")
		t.Setenv(envInsecureSkipVerify, "sometimes")
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{Config: testProviderConfig(schemaResp.Schema, nil)}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Invalid REMNAWAVE_INSECURE_SKIP_VERIFY")
	})

	t.Run("invalid proxy env", func(t *testing.T) {
		t.Setenv(envEndpoint, "https://panel.example.com")
		t.Setenv(envAPIToken, "token")
		t.Setenv(envProxyHeaders, "sometimes")
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{Config: testProviderConfig(schemaResp.Schema, nil)}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Invalid REMNAWAVE_PROXY_HEADERS")
	})

	t.Run("invalid timeout", func(t *testing.T) {
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":        "https://panel.example.com",
				"api_token":       "token",
				"request_timeout": "tomorrow",
			}),
		}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Invalid request_timeout")
	})

	t.Run("invalid endpoint", func(t *testing.T) {
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":  "ftp://panel.example.com",
				"api_token": "token",
			}),
		}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Client init failed")
	})
}

func testProviderConfig(providerSchema providerschema.Schema, values map[string]any) tfsdk.Config {
	attributeTypes := map[string]tftypes.Type{
		"endpoint":             tftypes.String,
		"api_token":            tftypes.String,
		"username":             tftypes.String,
		"password":             tftypes.String,
		"insecure_skip_verify": tftypes.Bool,
		"request_timeout":      tftypes.String,
		"proxy_headers":        tftypes.Bool,
	}
	rawValues := make(map[string]tftypes.Value, len(attributeTypes))
	for name, attributeType := range attributeTypes {
		value := any(nil)
		if configured, ok := values[name]; ok {
			value = configured
		}
		rawValues[name] = tftypes.NewValue(attributeType, value)
	}
	return tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: attributeTypes}, rawValues),
		Schema: providerSchema,
	}
}

func assertDiagnosticSummary(t *testing.T, diagnostics diag.Diagnostics, want string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Summary() == want {
			return
		}
	}
	t.Fatalf("diagnostics = %v, want summary %q", diagnostics, want)
}
