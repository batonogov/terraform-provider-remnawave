package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &RemnawaveProvider{}

type RemnawaveProvider struct {
	version string
}

type RemnawaveProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	APIToken           types.String `tfsdk:"api_token"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	RequestTimeout     types.String `tfsdk:"request_timeout"`
	ProxyHeaders       types.Bool   `tfsdk:"proxy_headers"`
	CustomHeaders      types.Map    `tfsdk:"custom_headers"`
}

const (
	envEndpoint           = "REMNAWAVE_ENDPOINT"
	envAPIToken           = "REMNAWAVE_API_TOKEN"
	envUsername           = "REMNAWAVE_USERNAME"
	envPassword           = "REMNAWAVE_PASSWORD"
	envInsecureSkipVerify = "REMNAWAVE_INSECURE_SKIP_VERIFY"
	envRequestTimeout     = "REMNAWAVE_REQUEST_TIMEOUT"
	envProxyHeaders       = "REMNAWAVE_PROXY_HEADERS"
	envCustomHeaders      = "REMNAWAVE_CUSTOM_HEADERS"
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RemnawaveProvider{version: version}
	}
}

func (p *RemnawaveProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "remnawave"
	resp.Version = p.version
}

func (p *RemnawaveProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Terraform provider for Remnawave — a proxy management panel built on Xray-core. Manage VPN users, nodes, hosts, squads, billing, and more as infrastructure-as-code.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Remnawave panel, e.g. https://panel.example.com. Can also be set via REMNAWAVE_ENDPOINT env var.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Pre-generated API token (JWT) for authentication. If set, username/password are ignored. Can also be set via REMNAWAVE_API_TOKEN env var.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Remnawave admin username for login. Can also be set via REMNAWAVE_USERNAME env var.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Remnawave admin password for login. Can also be set via REMNAWAVE_PASSWORD env var.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification (useful for self-signed certs). Can also be set via REMNAWAVE_INSECURE_SKIP_VERIFY env var.",
			},
			"request_timeout": schema.StringAttribute{
				Optional:    true,
				Description: "HTTP request timeout (e.g. 30s, 1m). Default: 30s. Can also be set via REMNAWAVE_REQUEST_TIMEOUT env var.",
			},
			"proxy_headers": schema.BoolAttribute{
				Optional:    true,
				Description: "Send X-Forwarded-For/X-Forwarded-Proto headers to bypass ProxyCheckMiddleware when connecting without a reverse proxy. Can also be set via REMNAWAVE_PROXY_HEADERS env var.",
			},
			"custom_headers": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "Custom HTTP headers to send with every request, for example reverse-proxy authentication headers. Can also be set as a JSON object via REMNAWAVE_CUSTOM_HEADERS env var. The HCL map takes precedence as a whole and is not merged with the environment value.",
			},
		},
	}
}

func (p *RemnawaveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config RemnawaveProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := envString(config.Endpoint, envEndpoint, "")
	if endpoint == "" {
		resp.Diagnostics.AddError("Missing endpoint", "endpoint must be set in the provider configuration or via REMNAWAVE_ENDPOINT env var.")
		return
	}

	apiToken := envString(config.APIToken, envAPIToken, "")
	username := envString(config.Username, envUsername, "")
	password := envString(config.Password, envPassword, "")

	if apiToken == "" && (username == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing credentials",
			"Either api_token or username+password must be provided.",
		)
		return
	}

	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() && !config.InsecureSkipVerify.IsUnknown() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	} else if v := os.Getenv(envInsecureSkipVerify); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddError("Invalid REMNAWAVE_INSECURE_SKIP_VERIFY", fmt.Sprintf("must be true or false, got %q", v))
			return
		}
		insecureSkipVerify = b
	}

	proxyHeaders := false
	if !config.ProxyHeaders.IsNull() && !config.ProxyHeaders.IsUnknown() {
		proxyHeaders = config.ProxyHeaders.ValueBool()
	} else if v := os.Getenv(envProxyHeaders); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddError("Invalid REMNAWAVE_PROXY_HEADERS", fmt.Sprintf("must be true or false, got %q", v))
			return
		}
		proxyHeaders = b
	}

	timeoutStr := envString(config.RequestTimeout, envRequestTimeout, "30s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		resp.Diagnostics.AddError("Invalid request_timeout", err.Error())
		return
	}

	customHeaders := make(map[string]string)
	if config.CustomHeaders.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unknown custom_headers",
			"custom_headers must be known during provider configuration. An unknown HCL value does not fall back to REMNAWAVE_CUSTOM_HEADERS.",
		)
		return
	}
	if !config.CustomHeaders.IsNull() {
		resp.Diagnostics.Append(config.CustomHeaders.ElementsAs(ctx, &customHeaders, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if rawHeaders := os.Getenv(envCustomHeaders); rawHeaders != "" {
		customHeaders, err = parseCustomHeadersEnv(rawHeaders)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid REMNAWAVE_CUSTOM_HEADERS",
				"REMNAWAVE_CUSTOM_HEADERS must be a non-null JSON object whose values are non-null strings. Header values are omitted from this diagnostic.",
			)
			return
		}
	}

	client, err := NewClient(ClientConfig{
		Endpoint:           endpoint,
		APIToken:           apiToken,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: insecureSkipVerify,
		Timeout:            timeout,
		ProxyHeaders:       proxyHeaders,
		CustomHeaders:      customHeaders,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client init failed", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RemnawaveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewNodeResource,
		NewHostResource,
		NewConfigProfileResource,
		NewSubscriptionSettingsResource,
		NewExternalSquadResource,
		NewInternalSquadResource,
		NewSubscriptionTemplateResource,
		NewPanelSettingsResource,
		NewSnippetResource,
		NewNodePluginResource,
		NewApiTokenResource,
		NewInfraProviderResource,
		NewBillingNodeResource,
		NewBillingHistoryResource,
		NewSubpageConfigResource,
		NewUserMetadataResource,
		NewNodeMetadataResource,
		NewHwidDeviceResource,
		NewHostBulkActionResource,
		NewNodeActionResource,
		NewDropConnectionsResource,
		NewUserActionResource,
		NewPasskeyResource,
		NewUserBulkActionResource,
		NewNodeBulkActionResource,
	}
}

func (p *RemnawaveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNodesDataSource,
		NewUsersDataSource,
		NewHostsDataSource,
		NewConfigProfilesDataSource,
		NewSystemHealthDataSource,
		NewKeygenDataSource,
		NewSubscriptionsDataSource,
		NewSubscriptionRequestHistoryDataSource,
		NewSystemStatsDataSource,
		NewSystemRecapDataSource,
		NewNodesMetricsDataSource,
		NewBandwidthStatsDataSource,
		NewBandwidthStatsUserDataSource,
		NewBandwidthRealtimeDataSource,
		NewSystemBandwidthStatsDataSource,
		NewSystemNodesStatsDataSource,
		NewSubscriptionRequestHistoryStatsDataSource,
		NewConnectionKeysDataSource,
		NewHwidStatsDataSource,
		NewHwidTopUsersDataSource,
		NewHostTagsDataSource,
		NewUserIPsDataSource,
		NewPasskeysDataSource,
		NewInternalSquadsDataSource,
		NewExternalSquadsDataSource,
	}
}

func parseCustomHeadersEnv(raw string) (map[string]string, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))

	root, err := decoder.Token()
	if err != nil || root != json.Delim('{') {
		return nil, fmt.Errorf("expected a non-null JSON object")
	}

	headers := make(map[string]string)
	seen := make(map[string]struct{})
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid JSON object")
		}
		name, ok := nameToken.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string header name")
		}

		foldedName := strings.ToLower(name)
		if _, duplicate := seen[foldedName]; duplicate {
			return nil, fmt.Errorf("duplicate header name")
		}
		seen[foldedName] = struct{}{}

		valueToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid value for header %q", name)
		}
		value, ok := valueToken.(string)
		if !ok {
			return nil, fmt.Errorf("header %q must have a non-null string value", name)
		}
		headers[name] = value
	}

	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, fmt.Errorf("invalid JSON object")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after JSON object")
	}

	return headers, nil
}

func envString(tfVal types.String, envKey, fallback string) string {
	if !tfVal.IsNull() && !tfVal.IsUnknown() {
		return tfVal.ValueString()
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
