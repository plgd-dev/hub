package mongodb

import (
	"fmt"
	"time"

	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
)

const minDuration = time.Millisecond * 100

type BulkWriteConfig struct {
	Timeout       time.Duration `yaml:"timeout"`
	ThrottleTime  time.Duration `yaml:"throttleTime"`
	DocumentLimit uint16        `yaml:"documentLimit"`
}

func (c *BulkWriteConfig) Validate() error {
	if c.Timeout <= minDuration {
		return fmt.Errorf("timeout('%v')", c.Timeout)
	}
	if c.ThrottleTime <= minDuration {
		return fmt.Errorf("throttleTime('%v')", c.ThrottleTime)
	}
	return nil
}

type Config struct {
	Mongo     pkgMongo.Config `yaml:",inline"`
	BulkWrite BulkWriteConfig `yaml:"bulkWrite"`
}

func (c *Config) Validate() error {
	if err := c.BulkWrite.Validate(); err != nil {
		return fmt.Errorf("bulkWrite.%w", err)
	}
	return c.Mongo.Validate()
}
