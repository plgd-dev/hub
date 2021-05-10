package validator

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/net/http/client"
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
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}
