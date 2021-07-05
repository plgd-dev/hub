package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/listener"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
)

type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
	UI      UIConfig      `yaml:"ui" json:"ui"`
}

func (c *Config) Validate() error {
	err := c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.Clients.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	err = c.UI.Validate()
	if err != nil {
		return fmt.Errorf("ui.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	HTTP HTTPConfig `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type HTTPConfig struct {
	Connection listener.Config `yaml:",inline" json:",inline"`
	//WebSocket     WebSocketConfig  `yaml:"webSocket" json:"webSocket"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
}

func (c *HTTPConfig) Validate() error {
	/*
		err := c.WebSocket.Validate()
		if err != nil {
			return fmt.Errorf("webSocket.%w", err)
		}
	*/
	err := c.Authorization.Validate()
	if err != nil {
		return fmt.Errorf("authorization.%w", err)
	}

	return c.Connection.Validate()
}

type WebSocketConfig struct {
	ReadLimit   int64         `yaml:"readLimit" json:"readLimit"`
	ReadTimeout time.Duration `yaml:"readTimeout" json:"readTimeout"`
}

func (c *WebSocketConfig) Validate() error {
	if c.ReadLimit <= 0 {
		return fmt.Errorf("readLimit('%v')", c.ReadLimit)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("readTimeout('%v')", c.ReadTimeout)
	}
	return nil
}

type ClientsConfig struct {
	GrpcGateway GrpcServerConfig `yaml:"grpcGateway" json:"grpcGateway"`
}

type GrpcServerConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *GrpcServerConfig) Validate() error {
	err := c.Connection.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return err
}

type EventBusConfig struct {
	GoPoolSize int               `yaml:"goPoolSize" json:"goPoolSize"`
	NATS       subscriber.Config `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	err := c.NATS.Validate()
	if err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

func (c *ClientsConfig) Validate() error {
	err := c.GrpcGateway.Validate()
	if err != nil {
		return fmt.Errorf("resourceAggregate.%w", err)
	}

	return nil
}

// OAuthClientConfig represents oauth configuration for user interface exposed via getOAuthConfiguration handler
type OAuthClientConfig struct {
	Domain             string `json:"domain" yaml:"domain"`
	ClientID           string `json:"clientID" yaml:"clientID"`
	Audience           string `json:"audience" yaml:"audience"`
	Scope              string `json:"scope" yaml:"scope"`
	HTTPGatewayAddress string `json:"httpGatewayAddress" yaml:"httpGatewayAddress"`
}

func (c *OAuthClientConfig) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain('%v')", c.Domain)
	}
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	if c.Audience == "" {
		return fmt.Errorf("audience('%v')", c.Audience)
	}
	if c.HTTPGatewayAddress == "" {
		return fmt.Errorf("httpGatewayAddress('%v')", c.HTTPGatewayAddress)
	}
	return nil
}

// UIConfig represents user interface configuration
type UIConfig struct {
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Directory   string            `json:"directory" yaml:"directory"`
	OAuthClient OAuthClientConfig `json:"oauthClient" yaml:"oauthClient"`
}

func (c *UIConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Directory == "" {
		return fmt.Errorf("directory('%v')", c.Directory)
	}
	err := c.OAuthClient.Validate()
	if err != nil {
		return fmt.Errorf("oauthClient.%w", err)
	}
	return err
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
