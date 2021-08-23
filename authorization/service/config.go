package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	natsClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/kit/config"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	err := c.Clients.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	err = c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	return nil
}

type APIsConfig struct {
	GRPC server.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Storage  StorageConfig  `yaml:"storage" json:"storage"`
	Eventbus EventBusConfig `yaml:"eventBus" json:"eventBus"`
}

func (c *ClientsConfig) Validate() error {
	err := c.Storage.Validate()
	if err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	return nil
}

type EventBusConfig struct {
	NATS natsClient.ConfigPublisher `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	err := c.NATS.Validate()
	if err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

type StorageConfig struct {
	OwnerClaim string         `yaml:"ownerClaim" json:"ownerClaim"`
	MongoDB    mongodb.Config `yaml:"mongoDB" json:"mongoDB"`
}

func (c *StorageConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	err := c.MongoDB.Validate()
	if err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
