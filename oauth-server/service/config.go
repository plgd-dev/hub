package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager"
)

// Config represents application configuration
type Config struct {
	Address string
	Listen  certManager.Config
}

func (c *Config) SetDefaults() {
}

func (c Config) String() string {
	return config.ToString(c)
}
