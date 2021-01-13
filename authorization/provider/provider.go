package provider

import (
	"context"

	"github.com/plgd-dev/cloud/authorization/oauth"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	AuthCodeURL(csrfToken string) string
}

// Config general configuration
type Config struct {
	Provider string       `envconfig:"PROVIDER" env:"PROVIDER" long:"provider" default:"generic"` // value which comes from the device during the sign-up ("apn")
	OAuth2   oauth.Config `envconfig:"OAUTH" env:"OAUTH" long:"oauth"`
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
