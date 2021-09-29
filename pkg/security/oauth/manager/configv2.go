package manager

import (
	"fmt"
	"net/url"
	"time"

	"github.com/plgd-dev/cloud/pkg/net/http/client"
	"golang.org/x/oauth2/clientcredentials"
)

type ConfigV2 struct {
	ClientID                    string        `yaml:"clientID" json:"clientID"`
	ClientSecret                string        `yaml:"clientSecretFile" json:"clientSecretFile"`
	Scopes                      []string      `yaml:"scopes" json:"scopes"`
	TokenURL                    string        `yaml:"tokenURL" json:"tokenURL"`
	Audience                    string        `yaml:"audience" json:"audience"`
	VerifyServiceTokenFrequency time.Duration `yaml:"verifyServiceTokenFrequency" json:"verifyServiceTokenFrequency"`
	HTTP                        client.Config `yaml:"http" json:"http"`
}

func (c *ConfigV2) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("clientSecretFile('%v')", c.ClientSecret)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("tokenURL('%v')", c.TokenURL)
	}
	if c.VerifyServiceTokenFrequency < 1 {
		return fmt.Errorf("verifyServiceTokenFrequency('%v')", c.VerifyServiceTokenFrequency)
	}
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

// ToClientCredentials converts to clientcredentials.Config
func (c ConfigV2) ToClientCredentials() clientcredentials.Config {
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
