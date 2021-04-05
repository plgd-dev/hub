package provider

import (
	"context"
	"crypto/tls"
	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/kit/security/certManager/client"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	AuthCodeURL(csrfToken string) string
}

// Config general configuration
type Config struct {
	Provider  string        `yaml:"provider" json:"provider" envconfig:"PROVIDER" env:"PROVIDER" default:"generic"` // value which comes from the device during the sign-up ("apn")
	OwnerClaim string       `yaml:"ownerClaim" json:"ownerClaim" envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	OAuth     oauth.Config  `yaml:"oauth" json:"oauth" envconfig:"OAUTH" env:"OAUTH`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS" env:"TLS"`
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
