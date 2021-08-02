package authorization

import (
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/authorization/oauth"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
)

// Config general configuration
type Config struct {
	Domain       string `yaml:"domain" json:"domain"`
	oauth.Config `yaml:",inline"`
	HTTP         client.Config `yaml:"http" json:"http"`
}

func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain('%v')", c.Domain)
	}
	err := c.Config.Validate()
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	err = c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}
