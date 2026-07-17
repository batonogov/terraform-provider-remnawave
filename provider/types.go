package provider

import "encoding/json"

// User maps to the Remnawave Users model.
// API: /api/users (POST create, PATCH update, DELETE /:uuid, GET /:uuid)
type User struct {
	UUID                   string         `json:"uuid,omitempty"`
	ID                     int64          `json:"id,omitempty"`
	ShortUUID              string         `json:"shortUuid,omitempty"`
	Username               string         `json:"username"`
	Status                 string         `json:"status,omitempty"`
	TrafficLimitBytes      int64          `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy   string         `json:"trafficLimitStrategy,omitempty"`
	ExpireAt               string         `json:"expireAt"`
	TrojanPassword         string         `json:"trojanPassword,omitempty"`
	VlessUUID              string         `json:"vlessUuid,omitempty"`
	SsPassword             string         `json:"ssPassword,omitempty"`
	Description            *string        `json:"description,omitempty"`
	Tag                    *string        `json:"tag,omitempty"`
	TelegramID             *int64         `json:"telegramId,omitempty"`
	Email                  *string        `json:"email,omitempty"`
	HwidDeviceLimit        *int64         `json:"hwidDeviceLimit,omitempty"`
	ExternalSquadUUID      *string        `json:"externalSquadUuid,omitempty"`
	ActiveInternalSquads   []UserSquadRef `json:"activeInternalSquads,omitempty"`
	LastTriggeredThreshold int64          `json:"lastTriggeredThreshold,omitempty"`
	SubRevokedAt           *string        `json:"subRevokedAt,omitempty"`
	LastTrafficResetAt     *string        `json:"lastTrafficResetAt,omitempty"`
	CreatedAt              string         `json:"createdAt,omitempty"`
	UpdatedAt              string         `json:"updatedAt,omitempty"`
	SubscriptionURL        string         `json:"subscriptionUrl,omitempty"`
	UserTraffic            *UserTraffic   `json:"userTraffic,omitempty"`
}

// UserTraffic contains the traffic counters and connection timestamps returned
// by the extended user API response.
type UserTraffic struct {
	UsedTrafficBytes         int64   `json:"usedTrafficBytes"`
	LifetimeUsedTrafficBytes int64   `json:"lifetimeUsedTrafficBytes"`
	OnlineAt                 *string `json:"onlineAt"`
	FirstConnectedAt         *string `json:"firstConnectedAt"`
	LastConnectedNodeUUID    *string `json:"lastConnectedNodeUuid"`
}

// UserSquadRef accepts the object returned by the API while encoding the UUID
// string expected by user create/update requests.
type UserSquadRef struct {
	UUID string `json:"uuid"`
	Name string `json:"name,omitempty"`
}

func (r *UserSquadRef) UnmarshalJSON(data []byte) error {
	var uuid string
	if err := json.Unmarshal(data, &uuid); err == nil {
		r.UUID = uuid
		return nil
	}
	type alias UserSquadRef
	return json.Unmarshal(data, (*alias)(r))
}

func (r UserSquadRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.UUID)
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
	LastStatusChange          *string            `json:"lastStatusChange,omitempty"`
	LastStatusMessage         *string            `json:"lastStatusMessage,omitempty"`
	IsTrafficTrackingActive   bool               `json:"isTrafficTrackingActive"`
	TrafficLimitBytes         *int64             `json:"trafficLimitBytes,omitempty"`
	TrafficUsedBytes          *int64             `json:"trafficUsedBytes,omitempty"`
	TrafficResetDay           *int               `json:"trafficResetDay,omitempty"`
	NotifyPercent             *int               `json:"notifyPercent,omitempty"`
	ViewPosition              int                `json:"viewPosition,omitempty"`
	CountryCode               string             `json:"countryCode,omitempty"`
	ConsumptionMultiplier     *float64           `json:"consumptionMultiplier,omitempty"`
	NodeConsumptionMultiplier *float64           `json:"nodeConsumptionMultiplier,omitempty"`
	Tags                      []string           `json:"tags,omitempty"`
	ConfigProfile             *NodeConfigProfile `json:"configProfile,omitempty"`
	ProviderUUID              *string            `json:"providerUuid,omitempty"`
	Provider                  json.RawMessage    `json:"provider,omitempty"`
	ActivePluginUUID          *string            `json:"activePluginUuid,omitempty"`
	System                    json.RawMessage    `json:"system,omitempty"`
	Versions                  json.RawMessage    `json:"versions,omitempty"`
	Note                      *string            `json:"note,omitempty"`
	UsersOnline               int                `json:"usersOnline,omitempty"`
	XrayUptime                float64            `json:"xrayUptime,omitempty"`
	CreatedAt                 string             `json:"createdAt,omitempty"`
	UpdatedAt                 string             `json:"updatedAt,omitempty"`
}

// NodeConfigProfile is the config profile assignment for a node.
type NodeConfigProfile struct {
	ActiveConfigProfileUUID string                     `json:"activeConfigProfileUuid"`
	ActiveInbounds          []NodeConfigProfileInbound `json:"activeInbounds"`
}

// NodeConfigProfileInbound accepts the full inbound object returned by node
// reads while encoding only its UUID for create/update requests.
type NodeConfigProfileInbound struct {
	UUID     string  `json:"uuid"`
	Tag      string  `json:"tag,omitempty"`
	Type     string  `json:"type,omitempty"`
	Network  *string `json:"network,omitempty"`
	Security *string `json:"security,omitempty"`
	Port     *int    `json:"port,omitempty"`
}

func (i *NodeConfigProfileInbound) UnmarshalJSON(data []byte) error {
	var uuid string
	if err := json.Unmarshal(data, &uuid); err == nil {
		i.UUID = uuid
		return nil
	}
	type alias NodeConfigProfileInbound
	return json.Unmarshal(data, (*alias)(i))
}

func (i NodeConfigProfileInbound) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.UUID)
}

// Host maps to the Remnawave Hosts model.
// API: /api/hosts (POST create, PATCH update, DELETE /:uuid, GET /:uuid)
type Host struct {
	UUID                         string          `json:"uuid,omitempty"`
	Remark                       string          `json:"remark"`
	Address                      string          `json:"address"`
	Port                         int             `json:"port"`
	Path                         *string         `json:"path,omitempty"`
	SNI                          *string         `json:"sni,omitempty"`
	HostHeader                   *string         `json:"host,omitempty"`
	ALPN                         *string         `json:"alpn,omitempty"`
	Fingerprint                  *string         `json:"fingerprint,omitempty"`
	IsDisabled                   bool            `json:"isDisabled,omitempty"`
	SecurityLayer                string          `json:"securityLayer,omitempty"`
	XHTTPExtraParams             json.RawMessage `json:"xhttpExtraParams,omitempty"`
	MuxParams                    json.RawMessage `json:"muxParams,omitempty"`
	SockoptParams                json.RawMessage `json:"sockoptParams,omitempty"`
	FinalMask                    json.RawMessage `json:"finalMask,omitempty"`
	ServerDescription            *string         `json:"serverDescription,omitempty"`
	Tags                         []string        `json:"tags,omitempty"`
	IsHidden                     bool            `json:"isHidden,omitempty"`
	OverrideSniFromAddress       bool            `json:"overrideSniFromAddress,omitempty"`
	KeepSniBlank                 bool            `json:"keepSniBlank,omitempty"`
	PinnedPeerCertSha256         *string         `json:"pinnedPeerCertSha256,omitempty"`
	VerifyPeerCertByName         *string         `json:"verifyPeerCertByName,omitempty"`
	VlessRouteID                 *int            `json:"vlessRouteId,omitempty"`
	ShuffleHost                  bool            `json:"shuffleHost,omitempty"`
	MihomoX25519                 bool            `json:"mihomoX25519,omitempty"`
	MihomoIPVersion              *string         `json:"mihomoIpVersion,omitempty"`
	Inbound                      *HostInbound    `json:"inbound,omitempty"`
	Nodes                        []string        `json:"nodes,omitempty"`
	XrayJsonTemplateUUID         *string         `json:"xrayJsonTemplateUuid,omitempty"`
	ExcludedInternalSquads       []string        `json:"excludedInternalSquads,omitempty"`
	ExcludeFromSubscriptionTypes []string        `json:"excludeFromSubscriptionTypes,omitempty"`
	ViewPosition                 int             `json:"viewPosition,omitempty"`
	CreatedAt                    string          `json:"createdAt,omitempty"`
	UpdatedAt                    string          `json:"updatedAt,omitempty"`
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
	RawInbound  any     `json:"rawInbound,omitempty"`
}

type ConfigProfileNode struct {
	UUID        string `json:"uuid,omitempty"`
	Name        string `json:"name,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// SubscriptionSettings is a singleton (GET/PATCH /api/subscription-settings).
type SubscriptionSettings struct {
	UUID                        string          `json:"uuid,omitempty"`
	ProfileTitle                *string         `json:"profileTitle,omitempty"`
	SupportLink                 *string         `json:"supportLink,omitempty"`
	ProfileUpdateInterval       *int            `json:"profileUpdateInterval,omitempty"`
	IsProfileWebpageURLEnabled  *bool           `json:"isProfileWebpageUrlEnabled,omitempty"`
	ServeJsonAtBaseSubscription *bool           `json:"serveJsonAtBaseSubscription,omitempty"`
	IsShowCustomRemarks         *bool           `json:"isShowCustomRemarks,omitempty"`
	HappAnnounce                *string         `json:"happAnnounce,omitempty"`
	HappRouting                 *string         `json:"happRouting,omitempty"`
	RandomizeHosts              *bool           `json:"randomizeHosts,omitempty"`
	CustomRemarks               json.RawMessage `json:"customRemarks,omitempty"`
	CustomResponseHeaders       json.RawMessage `json:"customResponseHeaders,omitempty"`
	ResponseRules               json.RawMessage `json:"responseRules,omitempty"`
	HwidSettings                json.RawMessage `json:"hwidSettings,omitempty"`
}

// InternalSquad maps to the Remnawave InternalSquad model.
type InternalSquad struct {
	UUID     string   `json:"uuid,omitempty"`
	Name     string   `json:"name"`
	Inbounds []string `json:"inbounds"`
}

// AccessibleNode represents a node accessible through an internal squad's inbounds.
type AccessibleNode struct {
	UUID              string   `json:"uuid"`
	NodeName          string   `json:"nodeName"`
	CountryCode       string   `json:"countryCode"`
	ConfigProfileUUID string   `json:"configProfileUuid"`
	ConfigProfileName string   `json:"configProfileName"`
	ActiveInbounds    []string `json:"activeInbounds"`
}

// InternalSquadAccessibleNodes wraps the response from GET /api/internal-squads/:uuid/accessible-nodes.
type InternalSquadAccessibleNodes struct {
	SquadUUID       string           `json:"squadUuid"`
	AccessibleNodes []AccessibleNode `json:"accessibleNodes"`
}

// ExternalSquad maps to the Remnawave ExternalSquad model.
type ExternalSquad struct {
	UUID                 string          `json:"uuid,omitempty"`
	Name                 string          `json:"name"`
	Templates            json.RawMessage `json:"templates,omitempty"`
	SubscriptionSettings json.RawMessage `json:"subscriptionSettings,omitempty"`
	HostOverrides        json.RawMessage `json:"hostOverrides,omitempty"`
	ResponseHeaders      json.RawMessage `json:"responseHeaders,omitempty"`
	HwidSettings         json.RawMessage `json:"hwidSettings,omitempty"`
	CustomRemarks        json.RawMessage `json:"customRemarks,omitempty"`
	SubpageConfigUUID    *string         `json:"subpageConfigUuid,omitempty"`
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
	Title   *string `json:"title"`
	LogoURL *string `json:"logoUrl"`
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
	UUID          string   `json:"uuid,omitempty"`
	Name          string   `json:"name"`
	ExpiresInDays int64    `json:"expiresInDays,omitempty"`
	ExpireAt      string   `json:"expireAt,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	Token         string   `json:"token,omitempty"`
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

// SubpageConfig maps to the Remnawave SubscriptionPageConfig model.
type SubpageConfig struct {
	UUID   string `json:"uuid,omitempty"`
	Name   string `json:"name"`
	Config any    `json:"config,omitempty"`
}

// Passkey maps to the Remnawave Passkey model.
// API: /api/passkeys (GET list), DELETE /api/passkeys/:id, PATCH /api/passkeys/:id
type Passkey struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}
