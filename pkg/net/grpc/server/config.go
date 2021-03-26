package server

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
)

type Config struct {
	Addr string        `yaml:"address" json:"address"`
	TLS  server.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address")
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
