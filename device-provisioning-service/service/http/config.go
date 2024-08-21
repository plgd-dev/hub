package http

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

type AuthorizationConfig struct {
	OwnerClaim       string `yaml:"ownerClaim" json:"ownerClaim"`
	validator.Config `yaml:",inline" json:",inline"`
}

func (c *AuthorizationConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	return c.Config.Validate()
}

type Config struct {
	Connection    listener.Config     `yaml:",inline" json:",inline"`
	Authorization AuthorizationConfig `yaml:"authorization" json:"authorization"`
	Server        server.Config       `yaml:",inline" json:",inline"`
}

func (c *Config) Validate() error {
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return c.Connection.Validate()
}
