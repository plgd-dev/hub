package nats

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

type ConfigV2 struct {
	URL     string        `yaml:"url" json:"url"  default:"nats://localhost:4222"`
	TLS     client.Config `yaml:"tls" json:"tls"`
	Options []nats.Option `yaml:"-" json:"-"`
}

func (c *ConfigV2) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	return nil
}
