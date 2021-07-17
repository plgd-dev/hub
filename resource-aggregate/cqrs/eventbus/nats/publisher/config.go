package publisher

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

type Config struct {
	URL       string        `yaml:"url" json:"url"  default:"nats://localhost:4222"`
	TLS       client.Config `yaml:"tls" json:"tls"`
	JetStream bool          `yaml:"jetstream" json:"jetstream"`
	Options   []nats.Option `yaml:"-" json:"-"`
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
