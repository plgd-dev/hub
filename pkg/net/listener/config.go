package listener

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
)

type Config struct {
	Addr string        `yaml:"address" json:"address"`
	TLS  server.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address('%v')", c.Addr)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
