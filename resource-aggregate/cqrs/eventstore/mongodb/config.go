package mongodb

import (
	"time"

	mongoCfg "github.com/plgd-dev/cloud/pkg/mongodb"
)

// Config provides Mongo DB configuration options
type Config struct {
	BatchSize       int             `yaml:"batchSize" json:"batchSize"`
	MaxPoolSize     uint64          `yaml:"maxPoolSize" json:"maxPoolSize"`
	MaxConnIdleTime time.Duration   `yaml:"maxConnIdleTime" json:"maxConnIdleTime"`
	Embedded        mongoCfg.Config `yaml:",inline" json:",inline"`

	marshalerFunc   MarshalerFunc   `yaml:"-"`
	unmarshalerFunc UnmarshalerFunc `yaml:"-"`
}

func (c *Config) Validate() error {
	return c.Embedded.Validate()
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
