package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
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
}

const (
	envEndpoint           = "REMNAWAVE_ENDPOINT"
	envAPIToken           = "REMNAWAVE_API_TOKEN"
	envUsername           = "REMNAWAVE_USERNAME"
	envPassword           = "REMNAWAVE_PASSWORD"
	envInsecureSkipVerify = "REMNAWAVE_INSECURE_SKIP_VERIFY"
	envRequestTimeout     = "REMNAWAVE_REQUEST_TIMEOUT"
	envProxyHeaders       = "REMNAWAVE_PROXY_HEADERS"
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
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
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

	client, err := NewClient(ClientConfig{
		Endpoint:           endpoint,
		APIToken:           apiToken,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: insecureSkipVerify,
		Timeout:            timeout,
		ProxyHeaders:       proxyHeaders,
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
	}
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
