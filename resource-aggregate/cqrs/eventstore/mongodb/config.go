package mongodb

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

// Config provides Mongo DB configuration options
type Config struct {
	URI             string        `yaml:"uri" json:"uri"`
	Database        string        `yaml:"database" json:"database"`
	BatchSize       int           `yaml:"batchSize" json:"batchSize"`
	MaxPoolSize     uint64        `yaml:"maxPoolSize" json:"maxPoolSize"`
	MaxConnIdleTime time.Duration `yaml:"maxConnIdleTime" json:"maxConnIdleTime"`
	TLS             client.Config `yaml:"tls" json:"tls"`

	marshalerFunc   MarshalerFunc   `yaml:"-"`
	unmarshalerFunc UnmarshalerFunc `yaml:"-"`
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

// Option provides the means to use function call chaining
type Option interface {
	apply(cfg *Config)
}

type MarshalerOpt struct {
	f MarshalerFunc
}

func (o MarshalerOpt) apply(cfg *Config) {
	cfg.marshalerFunc = o.f
}

// WithMarshaler provides the possibility to set an marshaling function for the config
func WithMarshaler(f MarshalerFunc) MarshalerOpt {
	return MarshalerOpt{
		f: f,
	}
}

type UnmarshalerOpt struct {
	f UnmarshalerFunc
}

func (o UnmarshalerOpt) apply(cfg *Config) {
	cfg.unmarshalerFunc = o.f
}

// WithUnmarshaler provides the possibility to set an unmarshaling function for the config
func WithUnmarshaler(f UnmarshalerFunc) UnmarshalerOpt {
	return UnmarshalerOpt{
		f: f,
	}
}
