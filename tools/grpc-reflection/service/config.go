package service

import (
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
)

type Config struct {
	Log  log.Config `yaml:"log" json:"log"`
	APIs APIsConfig `yaml:"apis" json:"apis"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`
}

type GRPCConfig struct {
	ReflectedServices []string `yaml:"reflectedServices" json:"reflectedServices"`
	server.BaseConfig `yaml:",inline" json:",inline"`
}

func (c *GRPCConfig) Validate() error {
	// Check if ReflectedServices is not empty
	if len(c.ReflectedServices) == 0 {
		return errors.New("reflectedServices cannot be empty")
	}

	// Check if each service name is not empty
	for _, service := range c.ReflectedServices {
		if service == "" {
			return errors.New("reflectedServices contains an empty service name")
		}
	}

	// Validate the embedded server.Config
	if err := c.BaseConfig.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	return nil
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
