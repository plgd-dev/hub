package service

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	httpServer "github.com/plgd-dev/hub/v2/pkg/net/http/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	grpcService "github.com/plgd-dev/hub/v2/snippet-service/service/grpc"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/updater"
)

type HTTPConfig struct {
	Addr   string            `yaml:"address" json:"address"`
	Server httpServer.Config `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC grpcService.Config `yaml:"grpc" json:"grpc"`
	HTTP HTTPConfig         `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type EventBusConfig struct {
	NATS natsClient.Config `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Storage                storeConfig.Config            `yaml:"storage" json:"storage"`
	OpenTelemetryCollector otelClient.Config             `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
	EventBus               EventBusConfig                `yaml:"eventBus" json:"eventBus"`
	ResourceUpdater        updater.ResourceUpdaterConfig `yaml:"resourceUpdater" json:"resourceUpdater"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	if err := c.EventBus.Validate(); err != nil {
		return fmt.Errorf("eventBus.%w", err)
	}
	if err := c.ResourceUpdater.Validate(); err != nil {
		return fmt.Errorf("resourceUpdater.%w", err)
	}
	return nil
}

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
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if _, err := uuid.Parse(c.HubID); err != nil {
		return fmt.Errorf("hubID('%v') - %w", c.HubID, err)
	}

	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
