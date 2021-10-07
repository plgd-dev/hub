package service

import (
	"fmt"

	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/mongodb"
	"github.com/plgd-dev/hub/pkg/net/grpc/server"
	natsClient "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/kit/v2/config"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
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
	Storage  StorageConfig  `yaml:"storage" json:"storage"`
	Eventbus EventBusConfig `yaml:"eventBus" json:"eventBus"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventBus.%w", err)
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

type StorageConfig struct {
	MongoDB mongodb.Config `yaml:"mongoDB" json:"mongoDB"`
}

func (c *StorageConfig) Validate() error {
	if err := c.MongoDB.Validate(); err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
