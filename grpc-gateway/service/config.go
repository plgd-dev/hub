package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	natsClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
)

type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
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
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`
}

type GRPCConfig struct {
	OwnerCacheExpiration time.Duration `yaml:"ownerCacheExpiration" json:"ownerCacheExpiration"`
	// deprecated by SubscribeToEvents.CreateSubscription.include_current_state
	SubscriptionCacheExpiration time.Duration `yaml:"subscriptionCacheExpiration" json:"subscriptionCacheExpiration"`
	SubscriptionBufferSize      int           `yaml:"subscriptionBufferSize" json:"subscriptionBufferSize"`
	server.Config               `yaml:",inline" json:",inline"`
}

func (c *GRPCConfig) Validate() error {
	if c.OwnerCacheExpiration <= 0 {
		return fmt.Errorf("ownerCacheExpiration('%v')", c.OwnerCacheExpiration)
	}
	if c.SubscriptionBufferSize < 0 {
		return fmt.Errorf("subscriptionBufferSize('%v')", c.SubscriptionBufferSize)
	}
	return c.Config.Validate()
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type AuthorizationServerConfig struct {
	OwnerClaim string        `yaml:"ownerClaim" json:"ownerClaim"`
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *AuthorizationServerConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	err := c.Connection.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return err
}

type ClientsConfig struct {
	AuthServer        AuthorizationServerConfig `yaml:"authorizationServer" json:"authorizationServer"`
	Eventbus          EventBusConfig            `yaml:"eventBus" json:"eventBus"`
	ResourceAggregate GrpcServerConfig          `yaml:"resourceAggregate" json:"resourceAggregate"`
	ResourceDirectory GrpcServerConfig          `yaml:"resourceDirectory" json:"resourceDirectory"`
}

type EventBusConfig struct {
	GoPoolSize int               `yaml:"goPoolSize" json:"goPoolSize"`
	NATS       natsClient.Config `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	err := c.NATS.Validate()
	if err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

func (c *ClientsConfig) Validate() error {
	err := c.AuthServer.Validate()
	if err != nil {
		return fmt.Errorf("authorizationServer.%w", err)
	}
	err = c.Eventbus.Validate()
	if err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	err = c.ResourceAggregate.Validate()
	if err != nil {
		return fmt.Errorf("resourceAggregate.%w", err)
	}
	err = c.ResourceDirectory.Validate()
	if err != nil {
		return fmt.Errorf("resourceDirectory.%w", err)
	}
	return nil
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

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
