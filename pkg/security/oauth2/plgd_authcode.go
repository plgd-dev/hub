package oauth2

import (
	"context"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewAuthCodePlgdProvider(config Config, httpClient *client.Client) *AuthCodePlgdProvider {
	return &AuthCodePlgdProvider{
		Config:     config,
		HTTPClient: httpClient,
	}
}

// PlgdProvider configuration with new http client
type AuthCodePlgdProvider struct {
	Config     Config
	HTTPClient *client.Client
}

// Exchange Auth Code for Access Token via OAuth
func (p *AuthCodePlgdProvider) Exchange(ctx context.Context, authorizationCode string, redirectURI string) (*Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())

	oauth := p.Config.ToDefaultOAuth2()
	if redirectURI != "" {
		oauth.RedirectURL = redirectURI
	}
	token, err := oauth.Exchange(ctx, authorizationCode)
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
func (p *AuthCodePlgdProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	restoredToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	oauth := p.Config.ToDefaultOAuth2()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())
	tokenSource := oauth.TokenSource(ctx, restoredToken)
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

func (p *AuthCodePlgdProvider) Close() {
	p.HTTPClient.Close()
}

func (p *AuthCodePlgdProvider) HTTP() *http.Client {
	return p.HTTPClient.HTTP()
}
