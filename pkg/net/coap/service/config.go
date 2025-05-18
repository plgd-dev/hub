package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
)

type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

// Config represents configuration of coap service
type Config struct {
	Addr              string                  `yaml:"address" json:"address"`
	Protocols         []Protocol              `yaml:"protocols" json:"protocols"`
	MaxMessageSize    uint32                  `yaml:"maxMessageSize" json:"maxMessageSize"`
	MessagePoolSize   uint32                  `yaml:"messagePoolSize" json:"messagePoolSize"`
	MessageQueueSize  int                     `yaml:"messageQueueSize" json:"messageQueueSize"`
	BlockwiseTransfer BlockwiseTransferConfig `yaml:"blockwiseTransfer" json:"blockwiseTransfer"`
	TLS               TLSConfig               `yaml:"tls" json:"tls"`
	InactivityMonitor *InactivityMonitor      `yaml:"inactivityMonitor,omitempty" json:"inactivityMonitor,omitempty"`
	KeepAlive         *KeepAlive              `yaml:"keepAlive,omitempty" json:"keepAlive,omitempty"`
}

func (c *Config) GetTimeout() time.Duration {
	if c.KeepAlive != nil {
		return c.KeepAlive.Timeout
	}
	return c.InactivityMonitor.Timeout
}

func (c *Config) validateKeepAliveAndInactivityMonitor() error {
	if c.InactivityMonitor != nil {
		if err := c.InactivityMonitor.Validate(); err != nil {
			return fmt.Errorf("inactivityMonitor.%w", err)
		}
	}
	if c.KeepAlive != nil {
		if err := c.KeepAlive.Validate(); err != nil {
			return fmt.Errorf("keepAlive.%w", err)
		}
	}
	if c.KeepAlive == nil && c.InactivityMonitor == nil {
		return errors.New("keepAlive or inactivityMonitor must be set")
	}
	return nil
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address('%v')", c.Addr)
	}
	if c.MaxMessageSize <= 64 {
		return fmt.Errorf("maxMessageSize('%v')", c.MaxMessageSize)
	}
	if len(c.Protocols) == 0 {
		return fmt.Errorf("protocols('%v')", c.Protocols)
	}
	for i := range len(c.Protocols) {
		switch c.Protocols[i] {
		case TCP, UDP:
		default:
			return fmt.Errorf("protocols[%v]('%v')", i, c.Protocols[i])
		}
	}
	if err := c.BlockwiseTransfer.Validate(); err != nil {
		return fmt.Errorf("blockwiseTransfer.%w", err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return c.validateKeepAliveAndInactivityMonitor()
}

type TLSConfig struct {
	Enabled                        *bool                    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	DisconnectOnExpiredCertificate bool                     `yaml:"disconnectOnExpiredCertificate" json:"disconnectOnExpiredCertificate"`
	Embedded                       certManagerServer.Config `yaml:",inline" json:",inline"`
}

// IsEnabled returns true if TLS is not set or it is enabled
func (c TLSConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

type KeepAlive struct {
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

func (c *KeepAlive) Validate() error {
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout('%v')", c.Timeout)
	}
	return nil
}

type InactivityMonitor struct {
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

func (c *InactivityMonitor) Validate() error {
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout('%v')", c.Timeout)
	}
	return nil
}

type BlockwiseTransferConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	SZX     string `yaml:"blockSize" json:"blockSize"`
}

func (c *BlockwiseTransferConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	switch strings.ToLower(c.SZX) {
	case "16", "32", "64", "128", "256", "512", "1024", "bert":
	default:
		return fmt.Errorf("blockSize('%v')", c.SZX)
	}
	return nil
}

func (c *TLSConfig) Validate() error {
	if !c.IsEnabled() {
		return nil
	}
	return c.Embedded.Validate()
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
