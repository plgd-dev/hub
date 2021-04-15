package mongodb

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

// Config provides Mongo DB configuration options
type Config struct {
	URI             string        `yaml:"uri" json:"uri" default:"mongodb://localhost:27017"`
	Database        string        `yaml:"database" json:"database" default:"eventStore"`
	BatchSize       int           `yaml:"batchSize" json:"batchSize" default:"16"`
	MaxPoolSize     uint64        `yaml:"maxPoolSize" json:"maxPoolSize" default:"16"`
	MaxConnIdleTime time.Duration `yaml:"maxConnIdleTime" json:"maxConnIdleTime" default:"240s"`
	TLS             client.Config `yaml:"tls" json:"tls"`

	marshalerFunc   MarshalerFunc       `yaml:"-" json:"-"`
	unmarshalerFunc UnmarshalerFunc     `yaml:"-" json:"-"`
	goroutinePoolGo GoroutinePoolGoFunc `yaml:"-" json:"-"`
}

func (c *Config) Validate() error {
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

type GoroutinePoolGoOpt struct {
	goroutinePoolGo GoroutinePoolGoFunc
}

func (o GoroutinePoolGoOpt) apply(cfg *Config) {
	cfg.goroutinePoolGo = o.goroutinePoolGo
}

func WithGoPool(goroutinePoolGo GoroutinePoolGoFunc) GoroutinePoolGoOpt {
	return GoroutinePoolGoOpt{
		goroutinePoolGo: goroutinePoolGo,
	}
}
