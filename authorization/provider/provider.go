package provider

import (
	"context"

	"go.uber.org/zap"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	AuthCodeURL(csrfToken string) string
	Close()
}

// New creates GitHub OAuth client
func New(config Config, logger *zap.Logger, responseMode, accessType, responseType string) (Provider, error) {
	switch config.Provider {
	case "github":
		return NewGitHubProvider(config, logger, responseMode, accessType, responseType)
	case "plgd":
		return NewPlgdProvider(config, logger, responseMode, accessType, responseType)
	default:
		return NewGenericProvider(config, logger, responseMode, accessType, responseType)
	}
}
