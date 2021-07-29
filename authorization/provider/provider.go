package provider

import (
	"context"

	"github.com/plgd-dev/cloud/pkg/log"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	AuthCodeURL(csrfToken string) string
	Close()
}

// New creates GitHub OAuth client
func New(config Config, logger log.Logger, ownerClaim, responseMode, accessType, responseType string) (Provider, error) {
	switch config.Provider {
	case "github":
		return NewGitHubProvider(config, logger, ownerClaim, responseMode, accessType, responseType)
	case "plgd":
		return NewPlgdProvider(config, logger, ownerClaim, responseMode, accessType, responseType)
	default:
		return NewGenericProvider(config, logger, ownerClaim, responseMode, accessType, responseType)
	}
}
