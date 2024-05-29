package http

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

type Config struct {
	Connection    listener.Config  `yaml:",inline" json:",inline"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
	Server        server.Config    `yaml:",inline" json:",inline"`
}

func (c *Config) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("connection.%w", err)
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}

	return nil
}
