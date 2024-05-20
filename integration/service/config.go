package service

import (
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

type Config struct {
	Log log.Config `yaml:"log" json:"log"`
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

func (c *Config) Validate() error {
	return nil
}
