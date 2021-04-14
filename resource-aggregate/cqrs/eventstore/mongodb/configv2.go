package mongodb

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

// Config provides Mongo DB configuration options
type ConfigV2 struct {
	URI             string        `yaml:"uri" json:"uri" default:"mongodb://localhost:27017"`
	Database        string        `yaml:"database" json:"database" default:"eventStore"`
	BatchSize       int           `yaml:"batchSize" json:"batchSize" default:"16"`
	MaxPoolSize     uint64        `yaml:"maxPoolSize" json:"maxPoolSize" default:"16"`
	MaxConnIdleTime time.Duration `yaml:"maxConnIdleTime" json:"maxConnIdleTime" default:"240s"`
	TLS             client.Config `yaml:"tls" json:"tls"`

	marshalerFunc   MarshalerFunc   `yaml:"-" json:"-"`
	unmarshalerFunc UnmarshalerFunc `yaml:"-" json:"-"`
}

func (c *ConfigV2) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("uri('%v')", c.URI)
	}
	if c.Database == "" {
		return fmt.Errorf("database('%v')", c.Database)
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
