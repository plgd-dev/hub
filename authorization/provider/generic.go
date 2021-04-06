package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/plgd-dev/cloud/pkg/net/http/client"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// NewGenericProvider creates OAuth client
func NewGenericProvider(config Config, logger *zap.Logger, responseMode, accessType, responseType string) (*GenericProvider, error) {
	config.ResponseMode = responseMode
	config.AccessType = accessType
	config.ResponseType = responseType
	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, err
	}
	oauth2 := config.Config.ToOAuth2()
	return &GenericProvider{
		Config:     config,
		OAuth2:     &oauth2,
		HTTPClient: httpClient,
	}, nil
}

// GenericProvider configuration with new http client
type GenericProvider struct {
	Config     Config
	OAuth2     *oauth2.Config
	HTTPClient *client.Client
}

// GetProviderName returns unique name of the provider
func (p *GenericProvider) GetProviderName() string {
	return p.Config.Provider
}

// AuthCodeURL returns URL for redirecting to the authentication web page
func (p *GenericProvider) AuthCodeURL(csrfToken string) string {
	return p.Config.Config.AuthCodeURL(csrfToken)
}

// LogoutURL to logout the user
func (p *GenericProvider) LogoutURL(returnTo string) string {
	URL, err := url.Parse(p.OAuth2.Endpoint.AuthURL + "/logout") // todo parse root path
	if err != nil {
		panic("invalid OAuthEndpoint configured")
	}

	parameters := url.Values{}
	parameters.Add("returnTo", returnTo)
	parameters.Add("client_id", p.OAuth2.ClientID)
	URL.RawQuery = parameters.Encode()

	return URL.String()
}

// Exchange Auth Code for Access Token via OAuth
func (p *GenericProvider) Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error) {
	if p.GetProviderName() != authorizationProvider {
		return nil, fmt.Errorf("unsupported authorization provider")
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient.HTTP())

	token, err := p.OAuth2.Exchange(ctx, authorizationCode)
	if err != nil {
		return nil, err
	}

	oauthClient := p.OAuth2.Client(ctx, token)
	resp, err := oauthClient.Get(p.OAuth2.Endpoint.AuthURL + "/userinfo")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var profile map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	userID, ok := profile[p.Config.OwnerClaim].(string)
	if !ok {
		return nil, fmt.Errorf("cannot determine owner claim %v", p.Config.OwnerClaim)
	}

	t := Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Owner:        userID,
	}
	return &t, nil
}

// Refresh gets new Access Token via OAuth.
func (p *GenericProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	restoredToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	tokenSource := p.OAuth2.TokenSource(ctx, restoredToken)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	oauthClient := p.OAuth2.Client(ctx, token)
	resp, err := oauthClient.Get(p.OAuth2.Endpoint.AuthURL + "/userinfo")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var profile map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	userID, ok := profile[p.Config.OwnerClaim].(string)
	if !ok {
		return nil, fmt.Errorf("cannot determine owner claim %v", p.Config.OwnerClaim)
	}
	return &Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Owner:        userID,
	}, nil
}

func (p *GenericProvider) Close() {
	p.HTTPClient.Close()
}
