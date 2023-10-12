package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/config"
	pkgConfig "github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	HubID   string        `yaml:"hubID" json:"hubId"`
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if _, err := uuid.Parse(c.HubID); err != nil {
		return fmt.Errorf("hubID('%v') - %w", c.HubID, err)
	}
	return nil
}

type APIsConfig struct {
	GRPC server.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Storage                StorageConfig     `yaml:"storage" json:"storage"`
	Eventbus               EventBusConfig    `yaml:"eventBus" json:"eventBus"`
	OpenTelemetryCollector otelClient.Config `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventBus.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
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

type StorageConfig = config.Config

// String return string representation of Config
func (c Config) String() string {
	return pkgConfig.ToString(c)
}
