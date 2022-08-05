package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
)

// Config represents application configuration
type Config struct {
	Log       log.Config    `yaml:"log" json:"log"`
	APIs      APIsConfig    `yaml:"apis" json:"apis"`
	Clients   ClientsConfig `yaml:"clients" json:"clients"`
	TaskQueue queue.Config  `yaml:"taskQueue" json:"taskQueue"`
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
	if err := c.TaskQueue.Validate(); err != nil {
		return fmt.Errorf("taskQueue.%w", err)
	}
	return nil
}

type APIsConfig struct {
	HTTP HTTPConfig `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type HTTPConfig struct {
	Connection    listener.Config  `yaml:",inline" json:",inline"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
	Server        server.Config    `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization('%w')", err)
	}
	return nil
}

type ClientsConfig struct {
	Eventbus               EventBusConfig                    `yaml:"eventBus" json:"eventBus"`
	GrpcGateway            GrpcGatewayConfig                 `yaml:"grpcGateway" json:"grpcGateway"`
	ResourceAggregate      ResourceAggregateConfig           `yaml:"resourceAggregate" json:"resourceAggregate"`
	Storage                StorageConfig                     `yaml:"storage" json:"storage"`
	Subscription           SubscriptionConfig                `yaml:"subscription" json:"subscription"`
	OpenTelemetryCollector http.OpenTelemetryCollectorConfig `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventBus.%w", err)
	}
	if err := c.GrpcGateway.Validate(); err != nil {
		return fmt.Errorf("grpcGateway.%w", err)
	}
	if err := c.ResourceAggregate.Validate(); err != nil {
		return fmt.Errorf("resourceAggregate.%w", err)
	}
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.Subscription.Validate(); err != nil {
		return fmt.Errorf("subscription.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
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

type GrpcGatewayConfig struct {
	Connection grpcClient.Config `yaml:"grpc" json:"grpc"`
}

func (c *GrpcGatewayConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ResourceAggregateConfig struct {
	Connection grpcClient.Config `yaml:"grpc" json:"grpc"`
}

func (c *ResourceAggregateConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type StorageConfig struct {
	MongoDB mongodb.Config `yaml:"mongoDB" json:"mongoDb"`
}

func (c *StorageConfig) Validate() error {
	if err := c.MongoDB.Validate(); err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}

type SubscriptionConfig struct {
	HTTP HTTPSubscriptionConfig `yaml:"http" json:"http"`
}

func (c *SubscriptionConfig) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type HTTPSubscriptionConfig struct {
	ReconnectInterval time.Duration   `yaml:"reconnectInterval" json:"reconnectInterval"`
	EmitEventTimeout  time.Duration   `yaml:"emitEventTimeout" json:"emitEventTimeout"`
	TLS               cmClient.Config `yaml:"tls" json:"tls"`
}

func (c *HTTPSubscriptionConfig) Validate() error {
	if c.ReconnectInterval <= 0 {
		return fmt.Errorf("reconnectInterval('%v')", c.ReconnectInterval)
	}
	if c.EmitEventTimeout <= 0 {
		return fmt.Errorf("emitEventTimeout('%v')", c.EmitEventTimeout)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

// Return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
