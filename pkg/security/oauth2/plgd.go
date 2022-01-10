package oauth2

import (
	"context"

	"github.com/plgd-dev/hub/v2/pkg/file"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewPlgdProvider(ctx context.Context, config Config, logger log.Logger) (*PlgdProvider, error) {
	config.ResponseMode = "query"
	config.AccessType = "offline"
	config.ResponseType = "code"

	clientSecret, err := file.Load(config.ClientSecretFile, make([]byte, 4096))
	if err != nil {
		return nil, err
	}

	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, err
	}
	oidcfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		return nil, err
	}

	oauth2 := config.ToOAuth2(oidcfg.AuthURL, oidcfg.TokenURL, string(clientSecret))

	return &PlgdProvider{
		Config:     config,
		OAuth2:     &oauth2,
		HTTPClient: httpClient,
		OpenID:     oidcfg,
	}, nil
}

// PlgdProvider configuration with new http client
type PlgdProvider struct {
	Config     Config
	OAuth2     *oauth2.Config
	HTTPClient *client.Client
	OpenID     openid.Config
}

// Exchange Auth Code for Access Token via OAuth
func (p *PlgdProvider) Exchange(ctx context.Context, authorizationCode string) (*Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())

	token, err := p.OAuth2.Exchange(ctx, authorizationCode)
	if err != nil {
		return nil, err
	}

	t := Token{
		AccessToken:  AccessToken(token.AccessToken),
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	return &t, nil
}

// Refresh gets new Access Token via OAuth.
func (p *PlgdProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	restoredToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())
	tokenSource := p.OAuth2.TokenSource(ctx, restoredToken)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return &Token{
		AccessToken:  AccessToken(token.AccessToken),
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}, nil
}

func (p *PlgdProvider) Close() {
	p.HTTPClient.Close()
}
