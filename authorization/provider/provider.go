package provider

import (
	"context"

	"golang.org/x/oauth2"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	AuthCodeURL(csrfToken string) string
}

type Endpoint struct {
	AuthURL  string `envconfig:"AUTH_URL" env:"AUTH_URL"`
	TokenURL string `envconfig:"TOKEN_URL" env:"AUTH_URL"`
}

type OAuthConfig struct {
	ClientID     string   `envconfig:"CLIENT_ID" env:"CLIENT_ID"`
	ClientSecret string   `envconfig:"CLIENT_SECRET" env:"CLIENT_SECRET"`
	Scopes       []string `envconfig:"SCOPES" env:"SCOPES"`
	Endpoint     Endpoint `envconfig:"ENDPOINT" env:"ENDPOINT"`
	Audience     string   `envconfig:"AUDIENCE" env:"AUDIENCE"`
	RedirectURL  string   `envconfig:"REDIRECT_URL" env:"REDIRECT_URL"`
	AccessType   string   `envconfig:"ACCESS_TYPE" env:"ACCESS_TYPE"`
	ResponseType string   `envconfig:"RESPONSE_TYPE" env:"RESPONSE_TYPE"`
	ResponseMode string   `envconfig:"RESPONSE_MODE" env:"RESPONSE_MODE"`
}

func (c OAuthConfig) ToOAuth2() oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Scopes:       c.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.Endpoint.AuthURL,
			TokenURL: c.Endpoint.TokenURL,
		},
	}
}

func (c OAuthConfig) AuthCodeURL(csrfToken string) string {
	aud := c.Audience
	accessType := c.AccessType
	responseType := c.ResponseType
	responseMode := c.ResponseMode

	opts := make([]oauth2.AuthCodeOption, 0, 2)

	if accessType != "" {
		opts = append(opts, oauth2.SetAuthURLParam("access_type", accessType))
	}
	if aud != "" {
		opts = append(opts, oauth2.SetAuthURLParam("audience", aud))
	}
	if responseType != "" {
		opts = append(opts, oauth2.SetAuthURLParam("response_type", responseType))
	}
	if responseMode != "" {
		opts = append(opts, oauth2.SetAuthURLParam("response_mode", responseMode))
	}
	auth := c.ToOAuth2()
	return (&auth).AuthCodeURL(csrfToken, opts...)
}

// Config general configuration
type Config struct {
	Provider string      `envconfig:"PROVIDER" env:"PROVIDER" default:"generic"` // value which comes from the device during the sign-up ("apn")
	OAuth2   OAuthConfig `envconfig:"OAUTH" env:"OAUTH"`
}

// New creates GitHub OAuth client
func New(config Config) Provider {
	switch config.Provider {
	case "github":
		return NewGitHubProvider(config)
	case "test":
		return NewTestProvider()
	case "auth0":
		return NewAuth0Provider(config)
	default:
		return NewGenericProvider(config)
	}
}
