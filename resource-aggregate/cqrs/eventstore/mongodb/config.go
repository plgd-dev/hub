package mongodb

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"
)

// Option provides the means to use function call chaining
type Option func(Config) Config

// WithMarshaler provides the possibility to set an marshaling function for the config
func WithMarshaler(f MarshalerFunc) Option {
	return func(cfg Config) Config {
		cfg.marshalerFunc = f
		return cfg
	}
}

// WithUnmarshaler provides the possibility to set an unmarshaling function for the config
func WithUnmarshaler(f UnmarshalerFunc) Option {
	return func(cfg Config) Config {
		cfg.unmarshalerFunc = f
		return cfg
	}
}

// WithTLS configures connection to use TLS
func WithTLS(cfg *tls.Config) Option {
	return func(c Config) Config {
		c.tlsCfg = cfg
		return c
	}
}

// Config provides Mongo DB configuration options
type Config struct {
	URI             string        `long:"uri" env:"URI" envconfig:"URI" default:"mongodb://localhost:27017"`
	DatabaseName    string        `long:"dbName" env:"DATABASE" envconfig:"DATABASE" default:"eventStore"`
	BatchSize       int           `long:"batchSize" env:"BATCH_SIZE" envconfig:"BATCH_SIZE" default:"16"`
	MaxPoolSize     uint64        `long:"maxPoolSize" env:"MAX_POOL_SIZE" envconfig:"MAX_POOL_SIZE" default:"16"`
	MaxConnIdleTime time.Duration `long:"maxConnIdleTime" env:"MAX_CONN_IDLE_TIME" envconfig:"MAX_CONN_IDLE_TIME" default:"240s"`
	tlsCfg          *tls.Config
	marshalerFunc   MarshalerFunc
	unmarshalerFunc UnmarshalerFunc
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
