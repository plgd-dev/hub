package clientcredentials

import (
	"fmt"
	"net/url"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"golang.org/x/oauth2/clientcredentials"
)

type Config struct {
	Authority        string              `yaml:"authority" json:"authority"`
	ClientID         string              `yaml:"clientID" json:"clientId"`
	ClientSecretFile urischeme.URIScheme `yaml:"clientSecretFile" json:"clientSecretFile"`
	Scopes           []string            `yaml:"scopes" json:"scopes"`
	TokenURL         string              `yaml:"-" json:"tokenUrl"`
	Audience         string              `yaml:"audience" json:"audience"`
	ClientSecret     string              `yaml:"-" json:"clientSecret"`
	HTTP             pkgTls.HTTPConfig   `yaml:"http" json:"http"`
}

func (c *Config) Validate() error {
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	if c.ClientSecretFile == "" {
		return fmt.Errorf("clientSecretFile('%v')", c.ClientSecretFile)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	clientSecret, err := c.ClientSecretFile.Read()
	if err != nil {
		return fmt.Errorf("clientSecretFile('%v')-%w", c.ClientSecretFile, err)
	}
	c.ClientSecret = string(clientSecret)
	return nil
}

func (c Config) ToClientCredentials(tokenURL, clientSecret string) clientcredentials.Config {
	v := make(url.Values)
	if c.Audience != "" {
		v.Set("audience", c.Audience)
	}
	return clientcredentials.Config{
		ClientID:       c.ClientID,
		ClientSecret:   clientSecret,
		Scopes:         c.Scopes,
		TokenURL:       tokenURL,
		EndpointParams: v,
	}
}

func (c Config) ToDefaultClientCredentials() clientcredentials.Config {
	return c.ToClientCredentials(c.TokenURL, c.ClientSecret)
}
