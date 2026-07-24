package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	if len(schemaResp.Schema.Attributes) != 8 {
		t.Fatalf("provider attributes = %d, want 8", len(schemaResp.Schema.Attributes))
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

	customHeaders, ok := schemaResp.Schema.Attributes["custom_headers"].(providerschema.MapAttribute)
	if !ok {
		t.Fatalf("custom_headers attribute type = %T", schemaResp.Schema.Attributes["custom_headers"])
	}
	if !customHeaders.Optional || customHeaders.Required || !customHeaders.Sensitive {
		t.Errorf("custom_headers must be an optional sensitive map: %#v", customHeaders)
	}
	if !customHeaders.ElementType.Equal(types.StringType) {
		t.Errorf("custom_headers element type = %s, want %s", customHeaders.ElementType, types.StringType)
	}
}

func TestProviderRegistersUniqueResources(t *testing.T) {
	t.Parallel()

	p := New("test")()
	factories := p.Resources(context.Background())
	if len(factories) != 26 {
		t.Fatalf("resources = %d, want 26", len(factories))
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
	if len(factories) != 25 {
		t.Fatalf("data sources = %d, want 25", len(factories))
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

func TestSubscriptionCredentialAttributesAreSensitive(t *testing.T) {
	t.Parallel()

	t.Run("user short UUID", func(t *testing.T) {
		var schemaResp resource.SchemaResponse
		NewUserResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
		assertSensitiveResourceStringAttribute(t, schemaResp, "short_uuid")
	})

	t.Run("subscription selector and response", func(t *testing.T) {
		var schemaResp datasource.SchemaResponse
		NewSubscriptionsDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
		assertSensitiveDataSourceStringAttribute(t, schemaResp, "short_uuid")
		assertSensitiveDataSourceStringAttribute(t, schemaResp, "response")
	})

	t.Run("connection keys response", func(t *testing.T) {
		var schemaResp datasource.SchemaResponse
		NewConnectionKeysDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
		assertSensitiveDataSourceStringAttribute(t, schemaResp, "response")
	})
}

func assertSensitiveResourceStringAttribute(t *testing.T, schemaResp resource.SchemaResponse, name string) {
	t.Helper()
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	attribute, ok := schemaResp.Schema.Attributes[name].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute type = %T", name, schemaResp.Schema.Attributes[name])
	}
	if !attribute.Sensitive {
		t.Errorf("%s must be sensitive", name)
	}
}

func assertSensitiveDataSourceStringAttribute(t *testing.T, schemaResp datasource.SchemaResponse, name string) {
	t.Helper()
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	attribute, ok := schemaResp.Schema.Attributes[name].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute type = %T", name, schemaResp.Schema.Attributes[name])
	}
	if !attribute.Sensitive {
		t.Errorf("%s must be sensitive", name)
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
		envCustomHeaders,
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
		t.Setenv(envCustomHeaders, `{"Cookie":"gateway=env-secret","X-Gateway-Token":"env-token"}`)

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
		if len(client.customHeaders) != 2 || client.customHeaders["Cookie"] != "gateway=env-secret" || client.customHeaders["X-Gateway-Token"] != "env-token" {
			t.Errorf("client custom headers were not loaded from the environment")
		}
		transport := client.httpClient.Transport.(*http.Transport)
		if !transport.TLSClientConfig.InsecureSkipVerify {
			t.Errorf("InsecureSkipVerify = false")
		}
	})

	t.Run("configured custom headers replace environment map", func(t *testing.T) {
		t.Setenv(envCustomHeaders, `{"X-Environment-Only":"environment-secret"}`)

		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":       "https://panel.example.com",
				"api_token":      "token",
				"custom_headers": map[string]string{"Cookie": "gateway=hcl-secret"},
			}),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Configure() diagnostics: %v", resp.Diagnostics)
		}
		client := resp.ResourceData.(*Client)
		if len(client.customHeaders) != 1 || client.customHeaders["Cookie"] != "gateway=hcl-secret" {
			t.Errorf("client custom headers were merged with the environment map")
		}
	})

	t.Run("configured empty custom headers suppress invalid environment", func(t *testing.T) {
		t.Setenv(envCustomHeaders, `{"Cookie":"do-not-leak"`)

		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":       "https://panel.example.com",
				"api_token":      "token",
				"custom_headers": map[string]string{},
			}),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Configure() diagnostics: %v", resp.Diagnostics)
		}
		if client := resp.ResourceData.(*Client); len(client.customHeaders) != 0 {
			t.Errorf("client custom headers count = %d, want 0", len(client.customHeaders))
		}
	})

	t.Run("unknown configured custom headers do not use environment", func(t *testing.T) {
		t.Setenv(envCustomHeaders, `{"Cookie":"environment-secret"}`)

		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":       "https://panel.example.com",
				"api_token":      "token",
				"custom_headers": tftypes.UnknownValue,
			}),
		}, &resp)
		assertDiagnosticSummary(t, resp.Diagnostics, "Unknown custom_headers")
		assertDiagnosticsDoNotContain(t, resp.Diagnostics, "environment-secret")
		if resp.ResourceData != nil || resp.DataSourceData != nil {
			t.Fatal("Configure() created a client from the environment for unknown custom_headers")
		}
	})

	t.Run("invalid custom headers environment", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			forbidden []string
		}{
			{
				name:      "invalid JSON",
				value:     `{"Cookie":"syntax-secret"`,
				forbidden: []string{`{"Cookie":"syntax-secret"`, "syntax-secret"},
			},
			{
				name:      "non-object root",
				value:     `["root-secret"]`,
				forbidden: []string{`["root-secret"]`, "root-secret"},
			},
			{
				name:  "null root",
				value: "null",
			},
			{
				name:      "null value",
				value:     `{"Cookie":null}`,
				forbidden: []string{`{"Cookie":null}`},
			},
			{
				name:      "non-string value",
				value:     `{"Cookie":{"secret":"nested-secret"}}`,
				forbidden: []string{`{"Cookie":{"secret":"nested-secret"}}`, "nested-secret"},
			},
			{
				name:      "exact duplicate name",
				value:     `{"Cookie":"exact-first-secret","Cookie":"exact-second-secret"}`,
				forbidden: []string{"exact-first-secret", "exact-second-secret"},
			},
			{
				name:      "case-insensitive duplicate name",
				value:     `{"Cookie":"case-first-secret","cookie":"case-second-secret"}`,
				forbidden: []string{"case-first-secret", "case-second-secret"},
			},
			{
				name:      "trailing JSON value",
				value:     `{"Cookie":"trailing-secret"} {"X-Other":"other-secret"}`,
				forbidden: []string{"trailing-secret", "other-secret"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Setenv(envCustomHeaders, tt.value)
				var resp frameworkprovider.ConfigureResponse
				p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
					Config: testProviderConfig(schemaResp.Schema, map[string]any{
						"endpoint":  "https://panel.example.com",
						"api_token": "token",
					}),
				}, &resp)
				assertDiagnosticSummary(t, resp.Diagnostics, "Invalid REMNAWAVE_CUSTOM_HEADERS")
				assertDiagnosticsDoNotContain(t, resp.Diagnostics, tt.forbidden...)
			})
		}
	})

	t.Run("null configured custom header value", func(t *testing.T) {
		t.Setenv(envCustomHeaders, `{"Cookie":"environment-secret"}`)
		var resp frameworkprovider.ConfigureResponse
		p.Configure(context.Background(), frameworkprovider.ConfigureRequest{
			Config: testProviderConfig(schemaResp.Schema, map[string]any{
				"endpoint":       "https://panel.example.com",
				"api_token":      "token",
				"custom_headers": map[string]any{"Cookie": nil},
			}),
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("Configure() succeeded with a null configured custom header value")
		}
		assertDiagnosticsDoNotContain(t, resp.Diagnostics, "environment-secret")
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
		"custom_headers":       tftypes.Map{ElementType: tftypes.String},
	}
	rawValues := make(map[string]tftypes.Value, len(attributeTypes))
	for name, attributeType := range attributeTypes {
		value := any(nil)
		if configured, ok := values[name]; ok {
			value = configured
		}
		rawValues[name] = testProviderAttributeValue(attributeType, value)
	}
	return tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: attributeTypes}, rawValues),
		Schema: providerSchema,
	}
}

func testProviderAttributeValue(attributeType tftypes.Type, value any) tftypes.Value {
	mapType, isMap := attributeType.(tftypes.Map)
	if !isMap || value == nil {
		return tftypes.NewValue(attributeType, value)
	}

	mapValues := make(map[string]tftypes.Value)
	switch configured := value.(type) {
	case map[string]string:
		for key, element := range configured {
			mapValues[key] = tftypes.NewValue(mapType.ElementType, element)
		}
	case map[string]any:
		for key, element := range configured {
			mapValues[key] = tftypes.NewValue(mapType.ElementType, element)
		}
	case map[string]tftypes.Value:
		mapValues = configured
	default:
		return tftypes.NewValue(attributeType, value)
	}

	return tftypes.NewValue(attributeType, mapValues)
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

func assertDiagnosticsDoNotContain(t *testing.T, diagnostics diag.Diagnostics, forbidden ...string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		text := diagnostic.Summary() + "\n" + diagnostic.Detail()
		for _, value := range forbidden {
			if value != "" && strings.Contains(text, value) {
				t.Errorf("diagnostic contains sensitive input %q: %s", value, text)
			}
		}
	}
}
