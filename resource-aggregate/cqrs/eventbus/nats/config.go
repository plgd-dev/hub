package nats

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/plgd-dev/kit/security/certManager/client"

	nats "github.com/nats-io/nats.go"
)

// Option provides the means to use function call chaining
type Option func(Config) Config

type Config struct {
	URL       string         `yaml:"url" json:"url" default:"nats://localhost:4222"`
	TLSConfig client.Config  `yaml:"tls" json:"tls"`
	Options   []nats.Option  `yaml:",omitempty"`
}

// String returns string representation of Config.
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

// WithTLS configures connection to use TLS
func WithTLS(cfg *tls.Config) Option {
	return func(c Config) Config {
		c.Options = append(c.Options, nats.Secure(cfg))
		return c
	}
}
