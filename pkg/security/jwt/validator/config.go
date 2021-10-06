package validator

import (
	"fmt"

	"github.com/plgd-dev/cloud/v2/pkg/net/http/client"
)

type Config struct {
	Authority string        `yaml:"authority" json:"authority"`
	Audience  string        `yaml:"audience" json:"audience"`
	HTTP      client.Config `yaml:"http" json:"http"`
}

func (c *Config) Validate() error {
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
	if c.Audience == "" {
		return fmt.Errorf("audience('%v')", c.Audience)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}
