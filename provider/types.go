package provider

// User maps to the Remnawave Users model.
// API: /api/users (POST create, PATCH update, DELETE /:uuid, GET /:uuid)
type User struct {
	UUID                 string  `json:"uuid,omitempty"`
	ID                   int64   `json:"id,omitempty"`
	ShortUUID            string  `json:"shortUuid,omitempty"`
	Username             string  `json:"username"`
	Status               string  `json:"status,omitempty"`
	TrafficLimitBytes    int64   `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy string  `json:"trafficLimitStrategy,omitempty"`
	ExpireAt             string  `json:"expireAt"`
	TrojanPassword       string  `json:"trojanPassword,omitempty"`
	VlessUUID            string  `json:"vlessUuid,omitempty"`
	SsPassword           string  `json:"ssPassword,omitempty"`
	Description          *string `json:"description,omitempty"`
	Tag                  *string `json:"tag,omitempty"`
	TelegramID           *int64  `json:"telegramId,omitempty"`
	Email                *string `json:"email,omitempty"`
	HwidDeviceLimit      *int64  `json:"hwidDeviceLimit,omitempty"`
	ExternalSquadUUID    *string `json:"externalSquadUuid,omitempty"`
	CreatedAt            string  `json:"createdAt,omitempty"`
	UpdatedAt            string  `json:"updatedAt,omitempty"`
	SubscriptionURL      string  `json:"subscriptionUrl,omitempty"`
}

// Node maps to the Remnawave Nodes model.
// API: /api/nodes (POST create, PATCH update, DELETE /:uuid, GET /:uuid)
type Node struct {
	UUID                      string             `json:"uuid,omitempty"`
	Name                      string             `json:"name"`
	Address                   string             `json:"address"`
	Port                      *int               `json:"port,omitempty"`
	ProxyURL                  *string            `json:"proxyUrl,omitempty"`
	IsConnected               bool               `json:"isConnected,omitempty"`
	IsDisabled                bool               `json:"isDisabled,omitempty"`
	IsConnecting              bool               `json:"isConnecting,omitempty"`
	IsTrafficTrackingActive   bool               `json:"isTrafficTrackingActive"`
	TrafficLimitBytes         *int64             `json:"trafficLimitBytes,omitempty"`
	TrafficUsedBytes          *int64             `json:"trafficUsedBytes,omitempty"`
	TrafficResetDay           *int               `json:"trafficResetDay,omitempty"`
	NotifyPercent             *int               `json:"notifyPercent,omitempty"`
	ViewPosition              int                `json:"viewPosition,omitempty"`
	CountryCode               string             `json:"countryCode,omitempty"`
	ConsumptionMultiplier     float64            `json:"consumptionMultiplier,omitempty"`
	NodeConsumptionMultiplier float64            `json:"nodeConsumptionMultiplier,omitempty"`
	Tags                      []string           `json:"tags,omitempty"`
	ConfigProfile             *NodeConfigProfile `json:"configProfile,omitempty"`
	ProviderUUID              *string            `json:"providerUuid,omitempty"`
	ActivePluginUUID          *string            `json:"activePluginUuid,omitempty"`
	Note                      *string            `json:"note,omitempty"`
	UsersOnline               int                `json:"usersOnline,omitempty"`
	XrayUptime                float64            `json:"xrayUptime,omitempty"`
	CreatedAt                 string             `json:"createdAt,omitempty"`
	UpdatedAt                 string             `json:"updatedAt,omitempty"`
}

// NodeConfigProfile is the config profile assignment for a node.
type NodeConfigProfile struct {
	ActiveConfigProfileUUID string   `json:"activeConfigProfileUuid"`
	ActiveInbounds          []string `json:"activeInbounds"`
}

// Host maps to the Remnawave Hosts model.
// API: /api/hosts (POST create, PATCH update, DELETE /:uuid, GET /:uuid)
type Host struct {
	UUID                   string       `json:"uuid,omitempty"`
	Remark                 string       `json:"remark"`
	Address                string       `json:"address"`
	Port                   int          `json:"port"`
	Path                   *string      `json:"path,omitempty"`
	SNI                    *string      `json:"sni,omitempty"`
	HostHeader             *string      `json:"host,omitempty"`
	ALPN                   *string      `json:"alpn,omitempty"`
	Fingerprint            *string      `json:"fingerprint,omitempty"`
	IsDisabled             bool         `json:"isDisabled,omitempty"`
	SecurityLayer          string       `json:"securityLayer,omitempty"`
	ServerDescription      *string      `json:"serverDescription,omitempty"`
	Tags                   []string     `json:"tags,omitempty"`
	IsHidden               bool         `json:"isHidden,omitempty"`
	OverrideSniFromAddress bool         `json:"overrideSniFromAddress,omitempty"`
	KeepSniBlank           bool         `json:"keepSniBlank,omitempty"`
	PinnedPeerCertSha256   *string      `json:"pinnedPeerCertSha256,omitempty"`
	VerifyPeerCertByName   *string      `json:"verifyPeerCertByName,omitempty"`
	VlessRouteID           *int         `json:"vlessRouteId,omitempty"`
	ShuffleHost            bool         `json:"shuffleHost,omitempty"`
	MihomoX25519           bool         `json:"mihomoX25519,omitempty"`
	MihomoIPVersion        *string      `json:"mihomoIpVersion,omitempty"`
	Inbound                *HostInbound `json:"inbound,omitempty"`
	Nodes                  []string     `json:"nodes,omitempty"`
	XrayJsonTemplateUUID   *string      `json:"xrayJsonTemplateUuid,omitempty"`
	ExcludedInternalSquads []string     `json:"excludedInternalSquads,omitempty"`
	ViewPosition           int          `json:"viewPosition,omitempty"`
	CreatedAt              string       `json:"createdAt,omitempty"`
	UpdatedAt              string       `json:"updatedAt,omitempty"`
}

// HostInbound links a host to a config profile inbound.
type HostInbound struct {
	ConfigProfileUUID        string `json:"configProfileUuid"`
	ConfigProfileInboundUUID string `json:"configProfileInboundUuid"`
}

// ConfigProfile maps to the Remnawave ConfigProfile model.
// API: /api/config-profiles (POST create, PATCH update, DELETE /:uuid, GET /:uuid, GET list)
type ConfigProfile struct {
	UUID   string `json:"uuid,omitempty"`
	Name   string `json:"name"`
	Config any    `json:"config,omitempty"`
	// Computed fields
	Inbounds []ConfigProfileInbound `json:"inbounds,omitempty"`
	Nodes    []ConfigProfileNode    `json:"nodes,omitempty"`
}

type ConfigProfileInbound struct {
	UUID        string  `json:"uuid,omitempty"`
	ProfileUUID string  `json:"profileUuid,omitempty"`
	Tag         string  `json:"tag,omitempty"`
	Type        string  `json:"type,omitempty"`
	Network     *string `json:"network,omitempty"`
	Security    *string `json:"security,omitempty"`
	Port        *int    `json:"port,omitempty"`
}

type ConfigProfileNode struct {
	UUID        string `json:"uuid,omitempty"`
	Name        string `json:"name,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// SubscriptionSettings is a singleton (GET/PATCH /api/subscription-settings).
type SubscriptionSettings struct {
	UUID                        string  `json:"uuid,omitempty"`
	ProfileTitle                *string `json:"profileTitle,omitempty"`
	SupportLink                 *string `json:"supportLink,omitempty"`
	ProfileUpdateInterval       *int    `json:"profileUpdateInterval,omitempty"`
	IsProfileWebpageURLEnabled  *bool   `json:"isProfileWebpageUrlEnabled,omitempty"`
	ServeJsonAtBaseSubscription *bool   `json:"serveJsonAtBaseSubscription,omitempty"`
	IsShowCustomRemarks         *bool   `json:"isShowCustomRemarks,omitempty"`
	HappAnnounce                *string `json:"happAnnounce,omitempty"`
	HappRouting                 *string `json:"happRouting,omitempty"`
	RandomizeHosts              *bool   `json:"randomizeHosts,omitempty"`
}

// InternalSquad maps to the Remnawave InternalSquad model.
type InternalSquad struct {
	UUID     string   `json:"uuid,omitempty"`
	Name     string   `json:"name"`
	Inbounds []string `json:"inbounds"`
}

// ExternalSquad maps to the Remnawave ExternalSquad model.
type ExternalSquad struct {
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name"`
}

// SubscriptionTemplate maps to the Remnawave SubscriptionTemplate model.
type SubscriptionTemplate struct {
	UUID                string `json:"uuid,omitempty"`
	Name                string `json:"name"`
	TemplateType        string `json:"templateType,omitempty"`
	TemplateJSON        any    `json:"templateJson,omitempty"`
	EncodedTemplateYaml string `json:"encodedTemplateYaml,omitempty"`
}

// PanelSettings maps to the Remnawave RemnawaveSettings model (singleton).
type PanelSettings struct {
	BrandingSettings *BrandingSettings     `json:"brandingSettings,omitempty"`
	PasswordSettings *PasswordAuthSettings `json:"passwordSettings,omitempty"`
	PasskeySettings  any                   `json:"passkeySettings,omitempty"`
	OAuth2Settings   any                   `json:"oauth2Settings,omitempty"`
}

type BrandingSettings struct {
	Title   *string `json:"title,omitempty"`
	LogoURL *string `json:"logoUrl,omitempty"`
}

type PasswordAuthSettings struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// Snippet maps to the Remnawave Snippet model (keyed by name, not UUID).
type Snippet struct {
	Name    string `json:"name"`
	Snippet any    `json:"snippet,omitempty"`
}

// NodePlugin maps to the Remnawave NodePlugin model.
type NodePlugin struct {
	UUID         string `json:"uuid,omitempty"`
	Name         string `json:"name"`
	PluginConfig any    `json:"pluginConfig,omitempty"`
}

// ApiToken maps to the Remnawave ApiToken model.
type ApiToken struct {
	UUID     string   `json:"uuid,omitempty"`
	Name     string   `json:"name"`
	ExpireAt string   `json:"expireAt,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	Token    string   `json:"token,omitempty"`
}

// InfraProvider maps to the Remnawave InfraProvider model.
type InfraProvider struct {
	UUID        string  `json:"uuid,omitempty"`
	Name        string  `json:"name"`
	FaviconLink *string `json:"faviconLink,omitempty"`
	LoginURL    *string `json:"loginUrl,omitempty"`
}

// BillingNode maps to the Remnawave InfraBillingNode model.
type BillingNode struct {
	UUID          string  `json:"uuid,omitempty"`
	ProviderUUID  string  `json:"providerUuid"`
	NodeUUID      *string `json:"nodeUuid,omitempty"`
	Name          *string `json:"name,omitempty"`
	NextBillingAt string  `json:"nextBillingAt"`
}

// BillingHistoryRecord maps to the Remnawave InfraBillingHistoryRecord model.
type BillingHistoryRecord struct {
	UUID         string  `json:"uuid,omitempty"`
	ProviderUUID string  `json:"providerUuid"`
	Amount       float64 `json:"amount"`
	BilledAt     string  `json:"billedAt"`
}
