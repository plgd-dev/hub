package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
	UI      UIConfig      `yaml:"ui" json:"ui"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.UI.Validate(); err != nil {
		return fmt.Errorf("ui.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	HTTP HTTPConfig `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type WebSocketConfig struct {
	StreamBodyLimit int           `yaml:"streamBodyLimit" json:"streamBodyLimit"`
	PingFrequency   time.Duration `yaml:"pingFrequency" json:"pingFrequency"`
}

func (c *WebSocketConfig) Validate() error {
	if c.StreamBodyLimit <= 0 {
		return fmt.Errorf("streamBodyLimit('%v')", c.StreamBodyLimit)
	}
	if c.PingFrequency <= 0 {
		return fmt.Errorf("pingFrequency('%v')", c.PingFrequency)
	}
	return nil
}

type HTTPConfig struct {
	Connection    listener.Config  `yaml:",inline" json:",inline"`
	WebSocket     WebSocketConfig  `yaml:"webSocket" json:"webSocket"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
	Server        server.Config    `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if err := c.WebSocket.Validate(); err != nil {
		return fmt.Errorf("webSocket.%w", err)
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return c.Connection.Validate()
}

type ClientsConfig struct {
	GrpcGateway            GrpcServerConfig                  `yaml:"grpcGateway" json:"grpcGateway"`
	OpenTelemetryCollector http.OpenTelemetryCollectorConfig `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

type GrpcServerConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *GrpcServerConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

func (c *ClientsConfig) Validate() error {
	if err := c.GrpcGateway.Validate(); err != nil {
		return fmt.Errorf("grpcGateway.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

type OAuthClient struct {
	Authority        string   `yaml:"authority" json:"authority,omitempty"`
	ClientID         string   `yaml:"clientID" json:"clientId"`
	Audience         string   `yaml:"audience" json:"audience"`
	Scopes           []string `yaml:"scopes" json:"scopes"`
	ProviderName     string   `json:"providerName" yaml:"providerName,omitempty"`
	GrantTypes       []string `json:"grantTypes" yaml:"grantTypes"`
	UseJWTPrivateKey bool     `json:"useJWTPrivateKey" yaml:"useJWTPrivateKey"`
}

func (c *OAuthClient) ToProto() *pb.OAuthClient {
	return &pb.OAuthClient{
		ClientId:         c.ClientID,
		Audience:         c.Audience,
		Scopes:           c.Scopes,
		ProviderName:     c.ProviderName,
		GrantTypes:       c.GrantTypes,
		UseJwtPrivateKey: c.UseJWTPrivateKey,
		Authority:        c.Authority,
	}
}

func (c *OAuthClient) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	return nil
}

type MainSidebarConfig struct {
	Devices              bool `yaml:"devices" json:"devices"`
	Configuration        bool `yaml:"configuration" json:"configuration"`
	RemoteClients        bool `yaml:"remoteClients" json:"remoteClients"`
	PendingCommands      bool `yaml:"pendingCommands" json:"pendingCommands"`
	Certificates         bool `yaml:"certificates" json:"certificates"`
	DeviceProvisioning   bool `yaml:"deviceProvisioning" json:"deviceProvisioning"`
	Docs                 bool `yaml:"docs" json:"docs"`
	ChatRoom             bool `yaml:"chatRoom" json:"chatRoom"`
	Dashboard            bool `yaml:"dashboard" json:"dashboard"`
	Integrations         bool `yaml:"integrations" json:"integrations"`
	DeviceFirmwareUpdate bool `yaml:"deviceFirmwareUpdate" json:"deviceFirmwareUpdate"`
	DeviceLogs           bool `yaml:"deviceLogs" json:"deviceLogs"`
	ApiTokens            bool `yaml:"apiTokens" json:"apiTokens"`
	SchemaHub            bool `yaml:"schemaHub" json:"schemaHub"`
}

func (c *MainSidebarConfig) ToProto() *pb.UIVisibility_MainSidebar {
	return &pb.UIVisibility_MainSidebar{
		Devices:              c.Devices,
		Configuration:        c.Configuration,
		RemoteClients:        c.RemoteClients,
		PendingCommands:      c.PendingCommands,
		Certificates:         c.Certificates,
		DeviceProvisioning:   c.DeviceProvisioning,
		Docs:                 c.Docs,
		ChatRoom:             c.ChatRoom,
		Dashboard:            c.Dashboard,
		Integrations:         c.Integrations,
		DeviceFirmwareUpdate: c.DeviceFirmwareUpdate,
		DeviceLogs:           c.DeviceLogs,
		ApiTokens:            c.ApiTokens,
		SchemaHub:            c.SchemaHub,
	}
}

type VisibilityConfig struct {
	MainSidebar MainSidebarConfig `yaml:"mainSidebar" json:"mainSidebar"`
}

func (c *VisibilityConfig) ToProto() *pb.UIVisibility {
	return &pb.UIVisibility{
		MainSidebar: c.MainSidebar.ToProto(),
	}
}

// WebConfiguration represents web configuration for user interface exposed via getOAuthConfiguration handler
type WebConfiguration struct {
	Authority                 string           `yaml:"-" json:"authority"`
	HTTPGatewayAddress        string           `yaml:"httpGatewayAddress" json:"httpGatewayAddress"`
	DeviceProvisioningService string           `yaml:"deviceProvisioningService" json:"deviceProvisioningService"`
	SnippetService            string           `yaml:"snippetService" json:"snippetService"`
	WebOAuthClient            OAuthClient      `yaml:"webOAuthClient" json:"webOauthClient"`
	DeviceOAuthClient         OAuthClient      `yaml:"deviceOAuthClient" json:"deviceOauthClient"`
	M2MOAuthClient            OAuthClient      `yaml:"m2mOAuthClient" json:"m2mOauthClient"`
	Visibility                VisibilityConfig `yaml:"visibility" json:"visibility"`
}

func (c *WebConfiguration) Validate() error {
	if c.HTTPGatewayAddress == "" {
		return fmt.Errorf("httpGatewayAddress('%v')", c.HTTPGatewayAddress)
	}
	if err := c.WebOAuthClient.Validate(); err != nil {
		return fmt.Errorf("webOAuthClient.%w", err)
	}
	if err := c.DeviceOAuthClient.Validate(); err != nil {
		return fmt.Errorf("deviceOAuthClient.%w", err)
	}
	return nil
}

// UIConfig represents user interface configuration
type UIConfig struct {
	Enabled          bool             `json:"enabled" yaml:"enabled"`
	Directory        string           `json:"directory" yaml:"directory"`
	WebConfiguration WebConfiguration `json:"webConfiguration" yaml:"webConfiguration"`
}

func (c *UIConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Directory == "" {
		return fmt.Errorf("directory('%v')", c.Directory)
	}
	if err := c.WebConfiguration.Validate(); err != nil {
		return fmt.Errorf("webConfiguration.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
