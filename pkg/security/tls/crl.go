package tls

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type HTTPConfigurer interface {
	GetMaxIdleConns() int
	GetMaxConnsPerHost() int
	GetMaxIdleConnsPerHost() int
	GetIdleConnTimeout() time.Duration
	GetTimeout() time.Duration
	GetTLS() ClientConfig

	Validate() error
}

type HTTPConfig struct {
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

	TLS ClientConfig `yaml:"tls" json:"tls"`
}

func (c *HTTPConfig) Validate() error {
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("maxIdleConns('%v')", c.MaxIdleConns)
	}
	if c.MaxConnsPerHost < 0 {
		return fmt.Errorf("maxConnsPerHost('%v')", c.MaxConnsPerHost)
	}
	if c.MaxIdleConnsPerHost < 0 {
		return fmt.Errorf("maxIdleConnsPerHost('%v')", c.MaxIdleConnsPerHost)
	}
	if c.IdleConnTimeout < 0 {
		return fmt.Errorf("idleConnTimeout('%v')", c.IdleConnTimeout)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout('%v')", c.Timeout)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

func (c *HTTPConfig) GetMaxIdleConns() int {
	return c.MaxIdleConns
}

func (c *HTTPConfig) GetMaxConnsPerHost() int {
	return c.MaxConnsPerHost
}

func (c *HTTPConfig) GetMaxIdleConnsPerHost() int {
	return c.MaxIdleConnsPerHost
}

func (c *HTTPConfig) GetIdleConnTimeout() time.Duration {
	return c.IdleConnTimeout
}

func (c *HTTPConfig) GetTimeout() time.Duration {
	return c.Timeout
}

func (c *HTTPConfig) GetTLS() ClientConfig {
	return c.TLS
}

type CRLConfig struct {
	Enabled bool           `yaml:"enabled" json:"enabled"`
	HTTP    HTTPConfigurer `yaml:"http" json:"http"`
}

func (c *CRLConfig) Equals(c2 CRLConfig) bool {
	if c.Enabled != c2.Enabled {
		return false
	}
	if !c.Enabled {
		r1 := c.HTTP == nil
		r2 := c2.HTTP == nil
		return r1 && r2
	}
	if c.HTTP == nil {
		return c2.HTTP == nil
	}
	tls := c.HTTP.GetTLS()
	return c.HTTP.GetMaxIdleConns() == c2.HTTP.GetMaxIdleConns() &&
		c.HTTP.GetMaxConnsPerHost() == c2.HTTP.GetMaxConnsPerHost() &&
		c.HTTP.GetMaxIdleConnsPerHost() == c2.HTTP.GetMaxIdleConnsPerHost() &&
		c.HTTP.GetIdleConnTimeout() == c2.HTTP.GetIdleConnTimeout() &&
		c.HTTP.GetTimeout() == c2.HTTP.GetTimeout() &&
		tls.Equals(c2.HTTP.GetTLS())
}

func (c *CRLConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.HTTP == nil {
		return errors.New("http configuration missing")
	}
	return c.HTTP.Validate()
}

func (c *CRLConfig) UnmarshalYAML(value *yaml.Node) error {
	type crlConfig struct {
		Enabled bool       `yaml:"enabled"`
		HTTP    HTTPConfig `yaml:"http"`
	}
	cc := crlConfig{}
	err := value.Decode(&cc)
	if err != nil {
		return err
	}
	c.Enabled = cc.Enabled
	if !cc.Enabled {
		c.HTTP = nil
		return nil
	}
	c.HTTP = &cc.HTTP
	return nil
}
