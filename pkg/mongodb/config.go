package mongodb

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
)

type Config struct {
	URI      string        `yaml:"uri" json:"uri"`
	Database string        `yaml:"database" json:"database"`
	TLS      client.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("uri('%v')", c.URI)
	}
	if c.Database == "" {
		return fmt.Errorf("database('%v')", c.Database)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
