package provider

import (
	"context"
	"crypto/tls"

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
	Provider string       `envconfig:"PROVIDER" env:"PROVIDER" default:"generic"` // value which comes from the device during the sign-up ("apn")
	OAuth2   oauth.Config `envconfig:"OAUTH" env:"OAUTH"`
}

// New creates GitHub OAuth client
func New(config Config, tls *tls.Config) Provider {
	switch config.Provider {
	case "github":
		return NewGitHubProvider(config)
	case "plgd":
		return NewPlgdProvider(config, tls)
	default:
		return NewGenericProvider(config)
	}
}
