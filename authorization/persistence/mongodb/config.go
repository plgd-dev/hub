package mongodb

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

type Config struct {
	URI      string        `yaml:"uri" json:"uri" default:"mongodb://localhost:27017"`
	Database string        `yaml:"database" json:"database" default:"authorization"`
	TLS      client.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("uri")
	}
	if c.Database == "" {
		return fmt.Errorf("database")
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
