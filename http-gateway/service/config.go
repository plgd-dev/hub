package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
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
	GrpcGateway GrpcServerConfig `yaml:"grpcGateway" json:"grpcGateway"`
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
	err := c.GrpcGateway.Validate()
	if err != nil {
		return fmt.Errorf("grpcGateway.%w", err)
	}

	return nil
}

type BasicOAuthClient struct {
	ClientID string   `yaml:"clientID" json:"clientId"`
	Audience string   `yaml:"audience" json:"audience"`
	Scopes   []string `yaml:"scopes" json:"scopes"`
}

func (c *BasicOAuthClient) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	return nil
}

type DeviceOAuthClient struct {
	BasicOAuthClient `yaml:",inline"`
	ProviderName     string `json:"providerName" yaml:"providerName"`
}

func (c *DeviceOAuthClient) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	return nil
}

// WebConfiguration represents web configuration for user interface exposed via getOAuthConfiguration handler
type WebConfiguration struct {
	Authority          string            `yaml:"authority" json:"authority"`
	HTTPGatewayAddress string            `yaml:"httpGatewayAddress" json:"httpGatewayAddress"`
	WebOAuthClient     BasicOAuthClient  `yaml:"webOAuthClient" json:"webOauthClient"`
	DeviceOAuthClient  DeviceOAuthClient `yaml:"deviceOAuthClient" json:"deviceOauthClient"`
}

func (c *WebConfiguration) Validate() error {
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
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

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
