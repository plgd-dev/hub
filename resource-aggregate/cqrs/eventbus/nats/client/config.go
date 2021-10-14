package client

import (
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/pkg/security/certManager/client"
)

type PendingLimitsConfig struct {
	MsgLimit   int `yaml:"msgLimit" json:"msgLimit"`
	BytesLimit int `yaml:"bytesLimit" json:"bytesLimit"`
}

func (c *PendingLimitsConfig) Validate() error {
	if c.MsgLimit == 0 {
		return fmt.Errorf("msgLimit('%v')", c.MsgLimit)
	}
	if c.BytesLimit == 0 {
		return fmt.Errorf("bytesLimit('%v')", c.BytesLimit)
	}
	return nil
}

type Config struct {
	URL            string              `yaml:"url" json:"url"`
	FlusherTimeout time.Duration       `yaml:"flusherTimeout" json:"flusherTimeout"`
	PendingLimits  PendingLimitsConfig `yaml:"pendingLimits" json:"pendingLimits"`
	TLS            client.Config       `yaml:"tls" json:"tls"`
	Options        []nats.Option       `yaml:"-" json:"-"`
}

type ConfigPublisher struct {
	JetStream bool `yaml:"jetstream" json:"jetstream"`
	Config    `yaml:",inline" json:",inline"`
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	if err := c.PendingLimits.Validate(); err != nil {
		return fmt.Errorf("pendingLimits.%w", err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

func (c *ConfigPublisher) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	if c.FlusherTimeout <= 0 {
		return fmt.Errorf("flusherTimeout('%v')", c.FlusherTimeout)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}
