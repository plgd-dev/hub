package oauth2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewClientCredentialsPlgdProvider(config Config, httpClient *client.Client, jwksURL string, ownerClaim, deviceIDClaim string, validator *jwt.Validator) *ClientCredentialsPlgdProvider {
	return &ClientCredentialsPlgdProvider{
		Config:        config,
		HTTPClient:    httpClient,
		ownerClaim:    ownerClaim,
		deviceIDClaim: deviceIDClaim,
		jwtValidator:  validator,
	}
}

// PlgdProvider configuration with new http client
type ClientCredentialsPlgdProvider struct {
	Config        Config
	HTTPClient    *client.Client
	ownerClaim    string
	deviceIDClaim string
	jwtValidator  *pkgJwt.Validator
}

func (p *ClientCredentialsPlgdProvider) parseToken(ctx context.Context, accessToken string) (pkgJwt.Claims, error) {
	if p.jwtValidator != nil {
		claims, err := p.jwtValidator.ParseWithContext(ctx, accessToken)
		if err != nil {
			return nil, fmt.Errorf("cannot verify authorization code: %w", err)
		}
		return pkgJwt.Claims(claims), nil
	}
	claims, err := pkgJwt.ParseToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("cannot parse authorization code: %w", err)
	}
	return pkgJwt.Claims(claims), nil
}

// Exchange Auth Code for Access Token via OAuth
func (p *ClientCredentialsPlgdProvider) Exchange(ctx context.Context, accessToken string) (*Token, error) {
	m, err := p.parseToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	c := p.Config.ToDefaultClientCredentials()
	if p.deviceIDClaim != "" {
		deviceID, errG := m.GetDeviceID(p.deviceIDClaim)
		if errG != nil {
			return nil, fmt.Errorf("cannot get deviceIDClaim: %w", errG)
		}
		if deviceID == "" {
			return nil, fmt.Errorf("deviceIDClaim('%v') is not set in token", p.deviceIDClaim)
		}
		c.EndpointParams.Add(p.deviceIDClaim, deviceID)
	}
	owner, err := m.GetOwner(p.ownerClaim)
	if err != nil {
		return nil, fmt.Errorf("cannot get ownerClaim: %w", err)
	}
	if owner == "" {
		return nil, fmt.Errorf("ownerClaim('%v') is not set in token", p.ownerClaim)
	}
	sub, err := m.GetSubject()
	if err != nil {
		return nil, fmt.Errorf("cannot get subject: %w", err)
	}
	c.EndpointParams.Add(p.ownerClaim, owner)
	if p.ownerClaim != "sub" {
		c.EndpointParams.Add("sub", sub)
	}
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
