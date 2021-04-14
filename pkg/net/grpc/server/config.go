package server

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
)

type Config struct {
	Addr          string           `yaml:"address" json:"address"`
	TLS           server.Config    `yaml:"tls" json:"tls"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address('%v')", c.Addr)
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	err = c.Authorization.Validate()
	if err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return nil
}
