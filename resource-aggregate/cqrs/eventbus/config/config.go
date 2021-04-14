package config

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
)

type Config struct {
	NATS nats.ConfigV2 `yaml:"nats" json:"nats"`
}

func (c *Config) Validate() error {
	err := c.NATS.Validate()
	if err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}
