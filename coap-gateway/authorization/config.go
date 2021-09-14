package authorization

import (
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/authorization/oauth"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
)

// Config general configuration
type Config struct {
	Authority    string `yaml:"authority" json:"authority"`
	oauth.Config `yaml:",inline"`
	HTTP         client.Config `yaml:"http" json:"http"`
}

func (c *Config) Validate() error {
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
	if err := c.Config.Validate(); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}
