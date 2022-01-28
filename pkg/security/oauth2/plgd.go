package oauth2

import (
	"context"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/file"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
)

type provider interface {
	Exchange(ctx context.Context, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	HTTP() *http.Client
	Close()
}

// NewPlgdProvider creates OAuth client
func NewPlgdProvider(ctx context.Context, config Config, logger log.Logger, ownerClaim, deviceIDClaim string) (*PlgdProvider, error) {
	config.ResponseMode = "query"
	config.AccessType = "offline"
	config.ResponseType = "code"

	clientSecret, err := file.Load(config.ClientSecretFile, make([]byte, 4096))
	if err != nil {
		return nil, err
	}
	config.ClientSecret = string(clientSecret)

	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, err
	}
	oidcfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		return nil, err
	}

	config.AuthURL = oidcfg.AuthURL
	config.TokenURL = oidcfg.TokenURL
	var provider provider
	if config.GrantType == oauth.ClientCredentials {
		provider = NewClientCredentialsPlgdProvider(config, httpClient, oidcfg.JWKSURL, ownerClaim, deviceIDClaim)
	} else {
		provider = NewAuthCodePlgdProvider(config, httpClient)
	}

	return &PlgdProvider{
		Config:   config,
		provider: provider,
		OpenID:   oidcfg,
	}, nil
}

// PlgdProvider configuration with new http client
type PlgdProvider struct {
	Config Config
	provider
	OpenID openid.Config
}
