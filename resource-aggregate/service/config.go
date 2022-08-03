package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	eventstoreConfig "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/config"
)

// Config represent application configuration
type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
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
	return nil
}

type APIsConfig struct {
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type GRPCConfig struct {
	OwnerCacheExpiration time.Duration `yaml:"ownerCacheExpiration" json:"ownerCacheExpiration"`
	grpcServer.Config    `yaml:",inline" json:",inline"`
}

func (c *GRPCConfig) Validate() error {
	if c.OwnerCacheExpiration <= 0 {
		return fmt.Errorf("ownerCacheExpiration('%v')", c.OwnerCacheExpiration)
	}
	return c.Config.Validate()
}

type EventBusConfig struct {
	NATS natsClient.ConfigPublisher `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

type EventStoreConfig struct {
	SnapshotThreshold            int                     `yaml:"snapshotThreshold" json:"snapshotThreshold"`
	ConcurrencyExceptionMaxRetry int                     `yaml:"occMaxRetry" json:"occMaxRetry"`
	DefaultCommandTimeToLive     time.Duration           `yaml:"defaultCommandTimeToLive" json:"defaultCommandTimeToLive"`
	Connection                   eventstoreConfig.Config `yaml:",inline" json:",inline"`
}

func (c *EventStoreConfig) Validate() error {
	if c.SnapshotThreshold <= 0 {
		return fmt.Errorf("snapshotThreshold('%v')", c.SnapshotThreshold)
	}
	if c.ConcurrencyExceptionMaxRetry <= 0 {
		return fmt.Errorf("occMaxRetry('%v')", c.ConcurrencyExceptionMaxRetry)
	}
	return c.Connection.Validate()
}

type IdentityStoreConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *IdentityStoreConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Eventbus               EventBusConfig      `yaml:"eventBus" json:"eventBus"`
	Eventstore             EventStoreConfig    `yaml:"eventStore" json:"eventStore"`
	IdentityStore          IdentityStoreConfig `yaml:"identityStore" json:"identityStore"`
	OpenTelemetryCollector otelClient.Config   `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	if err := c.Eventstore.Validate(); err != nil {
		return fmt.Errorf("eventstore.%w", err)
	}
	if err := c.IdentityStore.Validate(); err != nil {
		return fmt.Errorf("identityStore.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
