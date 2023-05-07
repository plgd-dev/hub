package oauth2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewClientCredentialsPlgdProvider(config Config, httpClient *client.Client, jwksURL string, ownerClaim, deviceIDClaim string) *ClientCredentialsPlgdProvider {
	keyCache := jwt.NewKeyCache(jwksURL, httpClient.HTTP())

	jwtValidator := jwt.NewValidator(keyCache)
	return &ClientCredentialsPlgdProvider{
		Config:        config,
		HTTPClient:    httpClient,
		ownerClaim:    ownerClaim,
		deviceIDClaim: deviceIDClaim,
		jwtValidator:  jwtValidator,
	}
}

// PlgdProvider configuration with new http client
type ClientCredentialsPlgdProvider struct {
	Config        Config
	HTTPClient    *client.Client
	ownerClaim    string
	deviceIDClaim string
	jwtValidator  *jwt.Validator
}

// Exchange Auth Code for Access Token via OAuth
func (p *ClientCredentialsPlgdProvider) Exchange(ctx context.Context, authorizationCode string) (*Token, error) {
	claims, err := p.jwtValidator.ParseWithContext(ctx, authorizationCode)
	if err != nil {
		return nil, fmt.Errorf("cannot verify authorization code: %w", err)
	}
	m := jwt.Claims(claims)
	c := p.Config.ToDefaultClientCredentials()
	if p.deviceIDClaim != "" {
		deviceID, err := m.GetDeviceID(p.deviceIDClaim)
		if err != nil {
			return nil, err
		}
		if deviceID == "" {
			return nil, fmt.Errorf("deviceIDClaim('%v') is not set in token", p.deviceIDClaim)
		}
		c.EndpointParams.Add(p.deviceIDClaim, deviceID)
	}
	owner, err := m.GetOwner(p.ownerClaim)
	if err != nil {
		return nil, err
	}
	if owner == "" {
		return nil, fmt.Errorf("ownerClaim('%v') is not set in token", p.ownerClaim)
	}
	c.EndpointParams.Add(p.ownerClaim, owner)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())
	token, err := c.Token(ctx)
	if err != nil {
		return nil, err
	}

	return &Token{
		AccessToken:  AccessToken(token.AccessToken),
		RefreshToken: token.AccessToken,
		Expiry:       token.Expiry,
	}, nil
}

// Refresh gets new Access Token via OAuth.
func (p *ClientCredentialsPlgdProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	return p.Exchange(ctx, refreshToken)
}

func (p *ClientCredentialsPlgdProvider) Close() {
	p.HTTPClient.Close()
}

func (p *ClientCredentialsPlgdProvider) HTTP() *http.Client {
	return p.HTTPClient.HTTP()
}
