package manager

import (
	"net/url"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

type Endpoint struct {
	TokenURL string `yaml:"tokenURL" json:"tokenURL" envconfig:"TOKEN_URL" env:"TOKEN_URL"`
}

type Config struct {
	ClientID       string        `yaml:"clientID" json:"clientID" envconfig:"CLIENT_ID" env:"CLIENT_ID"`
	ClientSecret   string        `yaml:"clientSecret" json:"clientSecret" envconfig:"CLIENT_SECRET" env:"CLIENT_SECRET"`
	Scopes         []string      `yaml:"scopes" json:"scopes" envconfig:"SCOPES" env:"SCOPES"`
	Endpoint                     `yaml:",inline"` // `envconfig:"ENDPOINT" env:"ENDPOINT"`
	Audience       string        `yaml:"audience" json:"audience" envconfig:"AUDIENCE" env:"AUDIENCE"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout" envconfig:"REQUEST_TIMEOUT" env:"REQUEST_TIMEOUT" default:"10s"`
	TickFrequency  time.Duration `yaml:"tick_frequency" json:"tick_frequency" envconfig:"TICK_FREQUENCY" env:"TICK_FREQUENCY" default:"15s" description:"how frequently we should check whether our token needs renewal"`
}


// ToClientCrendtials converts to clientcredentials.Config
func (c Config) ToClientCrendtials() clientcredentials.Config {
	v := make(url.Values)
	if c.Audience != "" {
		v.Set("audience", c.Audience)
	}

	return clientcredentials.Config{
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		Scopes:         c.Scopes,
		TokenURL:       c.Endpoint.TokenURL,
		EndpointParams: v,
	}
}
