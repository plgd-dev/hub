package service

import (
	"fmt"

	"github.com/plgd-dev/hub/pkg/config"
	"github.com/plgd-dev/hub/pkg/log"
	certManagerServer "github.com/plgd-dev/hub/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/pkg/sync/task/queue"
)

type Config struct {
	Log       LogConfig    `yaml:"log" json:"log"`
	APIs      APIsConfig   `yaml:"apis" json:"apis"`
	TaskQueue queue.Config `yaml:"taskQueue" json:"taskQueue"`
}

// Config represents application configuration
type LogConfig struct {
	log.Config       `yaml:",inline" json:",inline"`
	DumpCoapMessages bool `yaml:"dumpCoapMessages" json:"dumpCoapMessages"`
}

func (c *Config) Validate() error {
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.TaskQueue.Validate(); err != nil {
		return fmt.Errorf("taskQueue.%w", err)
	}
	return nil
}

type APIsConfig struct {
	COAP COAPConfig `yaml:"coap" json:"coap"`
}

func (c *APIsConfig) Validate() error {
	if err := c.COAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	return nil
}

type COAPConfig struct {
	Addr string    `yaml:"address" json:"address"`
	TLS  TLSConfig `yaml:"tls" json:"tls"`
}

func (c *COAPConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address('%v')", c.Addr)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

type TLSConfig struct {
	Enabled                  bool `yaml:"enabled" json:"enabled"`
	certManagerServer.Config `yaml:",inline" json:",inline"`
}

func (c Config) String() string {
	return config.ToString(c)
}
