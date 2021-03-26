package manager

import (
	"fmt"
	"net/url"
	"time"

	"github.com/plgd-dev/cloud/pkg/net/http/client"
	"golang.org/x/oauth2/clientcredentials"
)

type ConfigV2 struct {
	ClientID      string        `yaml:"clientID" json:"clientID"`
	ClientSecret  string        `yaml:"clientSecret" json:"clientSecret"`
	Scopes        []string      `yaml:"scopes" json:"scopes"`
	TokenURL      string        `yaml:"tokenURL" json:"tokenURL"`
	Audience      string        `yaml:"audience" json:"audience"`
	HTTP          client.Config `yaml:"http" json:"http"`
	TickFrequency time.Duration `yaml:"tickFrequency" json:"tickFrequency"`
}

func (c *ConfigV2) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("clientSecret('%v')", c.ClientSecret)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("tokenURL('%v')", c.TokenURL)
	}
	if c.TickFrequency < 1 {
		return fmt.Errorf("tickFrequency('%v')", c.TickFrequency)
	}
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

// ToClientCrendtials converts to clientcredentials.Config
func (c ConfigV2) ToClientCrendtials() clientcredentials.Config {
	v := make(url.Values)
	if c.Audience != "" {
		v.Set("audience", c.Audience)
	}

	return clientcredentials.Config{
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		Scopes:         c.Scopes,
		TokenURL:       c.TokenURL,
		EndpointParams: v,
	}
}
