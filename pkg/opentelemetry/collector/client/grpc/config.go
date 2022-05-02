package grpc

import "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"

type Config struct {
	Enabled    bool          `yaml:"enabled"`
	Connection client.Config `yaml:",inline"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	return nil
}
