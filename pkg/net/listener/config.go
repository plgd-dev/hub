package listener

import (
	"fmt"
	"net"

	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
)

type Config struct {
	Addr string        `yaml:"address" json:"address"`
	TLS  server.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
