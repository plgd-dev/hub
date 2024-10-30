package oauth2

import (
	"context"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"go.opentelemetry.io/otel/trace"
)

type provider interface {
	Exchange(ctx context.Context, authorizationCode string) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	HTTP() *http.Client
	Close()
}

// NewPlgdProvider creates OAuth client
func NewPlgdProvider(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, ownerClaim, deviceIDClaim string, validator *jwt.Validator) (*PlgdProvider, error) {
	config.ResponseMode = "query"
	config.AccessType = "offline"
	config.ResponseType = "code"

	clientSecret, err := config.ClientSecretFile.Read()
	if err != nil {
		return nil, err
	}
	config.ClientSecret = string(clientSecret)

	httpClient, err := cmClient.NewHTTPClient(&config.HTTP, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, err
	}
	oidcfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		return nil, err
	}

	config.AuthURL = oidcfg.AuthURL
	config.TokenURL = oidcfg.TokenURL
	var p provider
	if config.GrantType == oauth.ClientCredentials {
		p = NewClientCredentialsPlgdProvider(config, httpClient, ownerClaim, deviceIDClaim, validator)
	} else {
		p = NewAuthCodePlgdProvider(config, httpClient)
	}

	return &PlgdProvider{
		Config:   config,
		provider: p,
		OpenID:   oidcfg,
	}, nil
}

// PlgdProvider configuration with new http client
type PlgdProvider struct {
	Config Config
	provider
	OpenID openid.Config
}
