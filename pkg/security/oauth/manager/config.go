package manager

import (
	"net/url"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

type Endpoint struct {
	TokenURL string `envconfig:"TOKEN_URL" env:"TOKEN_URL"`
}

type Config struct {
	ClientID       string        `envconfig:"CLIENT_ID" env:"CLIENT_ID"`
	ClientSecret   string        `envconfig:"CLIENT_SECRET" env:"CLIENT_SECRET"`
	Scopes         []string      `envconfig:"SCOPES" env:"SCOPES"`
	Endpoint       Endpoint      `envconfig:"ENDPOINT" env:"ENDPOINT"`
	Audience       string        `envconfig:"AUDIENCE" env:"AUDIENCE"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" env:"REQUEST_TIMEOUT" default:"10s"`
	TickFrequency  time.Duration `envconfig:"TICK_FREQUENCY" env:"TICK_FREQUENCY" long:"tick-frequency" description:"how frequently we should check whether our token needs renewal" default:"15s"`
}

// ToClientCredentials converts to clientcredentials.Config
func (c Config) ToClientCredentials() clientcredentials.Config {
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
