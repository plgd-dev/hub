package provider

import (
	"fmt"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
)

// Config general configuration
type Config struct {
	Provider     string `yaml:"provider" json:"provider" default:"generic"` // value which comes from the device during the sign-up ("apn")
	oauth.Config `yaml:",inline"`
	HTTP         client.Config `yaml:"http" json:"http"`
	OwnerClaim   string        `yaml:"ownerClaim" json:"ownerClaim"`
}

func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider")
	}
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim")
	}
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	err = c.Config.Validate()
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
