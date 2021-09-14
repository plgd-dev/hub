package client

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

type KeepAliveConfig struct {
	// After a duration of this time if the client doesn't see any activity it
	// pings the server to see if the transport is still alive.
	// If set below 10s, a minimum value of 10s will be used instead.
	Time time.Duration `yaml:"time" json:"time"` // The current default value is infinity.
	// After having pinged for keepalive check, the client waits for a duration
	// of Timeout and if no activity is seen even after that the connection is
	// closed.
	Timeout time.Duration `yaml:"timeout" json:"timeout"` // The current default value is 20 seconds.
	// If true, client sends keepalive pings even with no active RPCs. If false,
	// when there are no active RPCs, Time and Timeout will be ignored and no
	// keepalive pings will be sent.
	PermitWithoutStream bool `yaml:"permitWithoutStream" json:"permitWithoutStream"` // false by default.
}

type Config struct {
	Addr      string          `yaml:"address" json:"address"`
	KeepAlive KeepAliveConfig `yaml:"keepAlive" json:"keepAlive"`
	TLS       client.Config   `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address('%v')", c.Addr)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
