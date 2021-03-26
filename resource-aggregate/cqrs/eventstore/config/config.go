package config

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
)

type Config struct {
	MongoDB mongodb.ConfigV2 `yaml:"mongoDB" json:"mongoDB"`
}

func (c *Config) Validate() error {
	err := c.MongoDB.Validate()
	if err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}
