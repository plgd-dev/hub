package authorization

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
	"github.com/plgd-dev/cloud/pkg/security/openid"
	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewPlgdProvider(ctx context.Context, config Config, logger log.Logger, ownerClaim, responseMode, accessType, responseType string) (*PlgdProvider, error) {
	config.ResponseMode = responseMode
	config.AccessType = accessType
	config.ResponseType = responseType
	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, err
	}
	oidcfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		return nil, err
	}

	oauth2 := config.Config.ToOAuth2(oidcfg.AuthURL, oidcfg.TokenURL)

	return &PlgdProvider{
		Config:     config,
		OAuth2:     &oauth2,
		HTTPClient: httpClient,
		OwnerClaim: ownerClaim,
		OpenID:     oidcfg,
	}, nil
}

// PlgdProvider configuration with new http client
type PlgdProvider struct {
	Config     Config
	OAuth2     *oauth2.Config
	HTTPClient *client.Client
	OwnerClaim string
	OpenID     openid.Config
}

// Exchange Auth Code for Access Token via OAuth
func (p *PlgdProvider) Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())

	token, err := p.OAuth2.Exchange(ctx, authorizationCode)
	if err != nil {
		return nil, err
	}

	owner, err := grpc.ParseOwnerFromJwtToken(p.OwnerClaim, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("configured owner claim '%v' is not present in the device token: %w", p.OwnerClaim, err)
	}

	t := Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Owner:        owner,
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

	owner, err := grpc.ParseOwnerFromJwtToken(p.OwnerClaim, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("configured owner claim '%v' is not present in the device token: %w", p.OwnerClaim, err)
	}

	return &Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Owner:        owner,
	}, nil
}

func (p *PlgdProvider) Close() {
	p.HTTPClient.Close()
}
