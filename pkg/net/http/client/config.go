package client

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
)

type Config struct {
	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	MaxIdleConns int `yaml:"maxIdleConns" json:"maxIdleConns"`

	// MaxConnsPerHost optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	MaxConnsPerHost int `yaml:"maxConnsPerHost" json:"maxConnsPerHost"`

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int `yaml:"maxIdleConnsPerHost" json:"maxIdleConnsPerHost"`

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration `yaml:"idleConnTimeout" json:"idleConnTimeout"`

	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time, any
	// redirects, and reading the response body. The timer remains
	// running after Get, Head, Post, or Do return and will
	// interrupt reading of the Response.Body.
	//
	// A Timeout of zero means no timeout.
	//
	// The Client cancels requests to the underlying Transport
	// as if the Request's Context ended.
	//
	// For compatibility, the Client will also use the deprecated
	// CancelRequest method on Transport if found. New
	// RoundTripper implementations should use the Request's Context
	// for cancellation instead of implementing CancelRequest.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	TLS client.Config `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("maxIdleConns")
	}
	if c.MaxConnsPerHost < 0 {
		return fmt.Errorf("maxConnsPerHost")
	}
	if c.MaxIdleConnsPerHost < 0 {
		return fmt.Errorf("maxIdleConnsPerHost")
	}
	if c.IdleConnTimeout < 0 {
		return fmt.Errorf("idleConnTimeout")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout")
	}
	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
