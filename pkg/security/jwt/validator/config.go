package validator

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/net/http/client"
)

type Config struct {
	URL  string        `yaml:"url" json:"url"`
	HTTP client.Config `yaml:"http" json:"http"`
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}
