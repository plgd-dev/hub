package config

import (
	"fmt"

	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore/mongodb"
)

type Config struct {
	MongoDB mongodb.Config `yaml:"mongoDB" json:"mongoDb"`
}

func (c *Config) Validate() error {
	if err := c.MongoDB.Validate(); err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}
