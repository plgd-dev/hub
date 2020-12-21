package jetstream

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/plgd-dev/cqrs/event"
)

// Option provides the means to use function call chaining
type Option func(Config) Config

// WithMarshaler provides the possibility to set an marshaling function for the config
func WithMarshaler(f event.MarshalerFunc) Option {
	return func(cfg Config) Config {
		cfg.marshalerFunc = f
		return cfg
	}
}

// WithUnmarshaler provides the possibility to set an unmarshaling function for the config
func WithUnmarshaler(f event.UnmarshalerFunc) Option {
	return func(cfg Config) Config {
		cfg.unmarshalerFunc = f
		return cfg
	}
}

type LogDebugFunc = func(fmt string, args ...interface{})

func WithLogDebug(f LogDebugFunc) Option {
	return func(cfg Config) Config {
		cfg.logDebug = f
		return cfg
	}
}

// Config provides NATS stream configuration options
type Config struct {
	URL     string `envconfig:"URL" default:"nats://localhost:4223"`
	Options []nats.Option
	// The minimum number of connections in the connection pool.
	InitialCap int `envconfig:"INITIAL_CAP" default:"8"`
	// Maximum number of concurrent surviving connections.
	MaxCap int `envconfig:"MAX_CAP" default:"24"`
	// Maximum idle connections.
	MaxIdle int `envconfig:"MAX_IDLE" default:"16"`
	// The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
	IdleTimeout time.Duration `envconfig:"IDLE_TIMEOUT" default:"10s"`

	marshalerFunc   event.MarshalerFunc
	unmarshalerFunc event.UnmarshalerFunc
	logDebug        LogDebugFunc
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
