package client

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
)

type GRPCConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Connection client.Config `yaml:",inline"`
}

func (c *GRPCConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	return nil
}

type Config struct {
	GRPC GRPCConfig `yaml:"grpc"`
}

func (c *Config) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}
