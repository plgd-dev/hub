package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	grpcClient "github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/listener"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	natsClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
)

// Config represents application configuration
type Config struct {
	Log           log.Config          `yaml:"log" json:"log"`
	APIs          APIsConfig          `yaml:"apis" json:"apis"`
	Clients       ClientsConfig       `yaml:"clients" json:"clients"`
	TaskProcessor TaskProcessorConfig `yaml:"taskProcessor" json:"taskProcessor"`
}

func (c *Config) Validate() error {
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.TaskProcessor.Validate(); err != nil {
		return fmt.Errorf("taskProcessor.%w", err)
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
	EventsURL     string            `yaml:"eventsURL" json:"eventsURL"`
	PullDevices   PullDevicesConfig `yaml:"pullDevices" json:"pullDevices"`
	Connection    listener.Config   `yaml:",inline" json:",inline"`
	Authorization oauth2.Config     `yaml:"authorization" json:"authorization"`
}

type PullDevicesConfig struct {
	Disabled bool          `yaml:"disabled" json:"disabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

func (c *PullDevicesConfig) Validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("interval('%v')", c.Interval)
	}
	return nil
}

func (c *HTTPConfig) Validate() error {
	if c.EventsURL == "" {
		return fmt.Errorf("eventsURL('%v')", c.EventsURL)
	}
	if err := c.PullDevices.Validate(); err != nil {
		return fmt.Errorf("pullDevices('%w')", err)
	}
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization('%w')", err)
	}
	return nil
}

type ClientsConfig struct {
	AuthServer        AuthorizationServerConfig `yaml:"authorizationServer" json:"authorizationServer"`
	Eventbus          EventBusConfig            `yaml:"eventBus" json:"eventBus"`
	ResourceAggregate ResourceAggregateConfig   `yaml:"resourceAggregate" json:"resourceAggregate"`
	ResourceDirectory ResourceDirectoryConfig   `yaml:"resourceDirectory" json:"resourceDirectory"`
	Storage           StorageConfig             `yaml:"storage" json:"storage"`
	Subscription      SubscriptionConfig        `yaml:"subscription" json:"subscription"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.AuthServer.Validate(); err != nil {
		return fmt.Errorf("authorizationServer.%w", err)
	}
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventBus.%w", err)
	}
	if err := c.ResourceAggregate.Validate(); err != nil {
		return fmt.Errorf("resourceAggregate.%w", err)
	}
	if err := c.ResourceDirectory.Validate(); err != nil {
		return fmt.Errorf("resourceDirectory.%w", err)
	}
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.Subscription.Validate(); err != nil {
		return fmt.Errorf("subscription.%w", err)
	}
	return nil
}

type AuthorizationServerConfig struct {
	Connection grpcClient.Config `yaml:"grpc" json:"grpc"`
}

func (c *AuthorizationServerConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
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

type ResourceAggregateConfig struct {
	Connection grpcClient.Config `yaml:"grpc" json:"grpc"`
}

func (c *ResourceAggregateConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ResourceDirectoryConfig struct {
	Connection grpcClient.Config `yaml:"grpc" json:"grpc"`
}

func (c *ResourceDirectoryConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type StorageConfig struct {
	MongoDB mongodb.Config `yaml:"mongoDB" json:"mongoDB"`
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
	ReconnectInterval   time.Duration `yaml:"reconnectInterval" json:"reconnectInterval"`
	ResubscribeInterval time.Duration `yaml:"resubscribeInterval" json:"resubscribeInterval"`
}

func (c *HTTPSubscriptionConfig) Validate() error {
	if c.ReconnectInterval <= 0 {
		return fmt.Errorf("reconnectInterval('%v')", c.ReconnectInterval)
	}
	if c.ResubscribeInterval <= 0 {
		return fmt.Errorf("resubscribeInterval('%v')", c.ResubscribeInterval)
	}
	return nil
}

type TaskProcessorConfig struct {
	CacheSize   int           `yaml:"cacheSize" json:"cacheSize"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	MaxParallel int           `yaml:"maxParallel" json:"maxParallel"`
	Delay       time.Duration `yaml:"delay" json:"delay"` // Used for CTT test with 10s.
}

func (c *TaskProcessorConfig) Validate() error {
	if c.CacheSize <= 0 {
		return fmt.Errorf("cacheSize('%v')", c.CacheSize)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout('%v')", c.Timeout)
	}
	if c.MaxParallel <= 0 {
		return fmt.Errorf("maxParallel('%v')", c.MaxParallel)
	}
	return nil
}

// Return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
