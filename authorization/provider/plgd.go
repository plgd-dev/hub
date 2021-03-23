package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// NewPlgdProvider creates OAuth client
func NewPlgdProvider(config Config, tls *tls.Config) *PlgdProvider {
	oauth2 := config.OAuth2.ToOAuth2()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.IdleConnTimeout = time.Second * 30
	t.TLSClientConfig = tls

	httpClient := &http.Client{
		Transport: t,
		Timeout:   10 * time.Minute,
	}
	return &PlgdProvider{
		Config: config,
		OAuth2: &oauth2,
		NewHTTPClient: func() *http.Client {
			return httpClient
		},
		TLS: tls,
	}
}

// PlgdProvider configuration with new http client
type PlgdProvider struct {
	Config        Config
	OAuth2        *oauth2.Config
	NewHTTPClient func() *http.Client
	TLS           *tls.Config
}

// GetProviderName returns unique name of the provider
func (p *PlgdProvider) GetProviderName() string {
	return p.Config.Provider
}

// AuthCodeURL returns URL for redirecting to the authentication web page
func (p *PlgdProvider) AuthCodeURL(csrfToken string) string {
	return p.Config.OAuth2.AuthCodeURL(csrfToken)
}

// LogoutURL to logout the user
func (p *PlgdProvider) LogoutURL(returnTo string) string {
	URL, err := url.Parse(p.OAuth2.Endpoint.AuthURL + "/v2/logout") // todo parse root path
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
func (p *PlgdProvider) Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*Token, error) {
	if p.GetProviderName() != authorizationProvider {
		return nil, fmt.Errorf("unsupported authorization provider(%v), only (%v) is supported", authorizationProvider, p.GetProviderName())
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.NewHTTPClient())

	token, err := p.OAuth2.Exchange(ctx, authorizationCode)
	if err != nil {
		return nil, err
	}

	oauthClient := p.OAuth2.Client(ctx, token)
	resp, err := oauthClient.Get(p.OAuth2.Endpoint.AuthURL + "/userinfo") // todo parse root path
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var profile map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	fmt.Printf("UserInfo: %+v\n", profile)

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
func (p *PlgdProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	restoredToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.NewHTTPClient())
	tokenSource := p.OAuth2.TokenSource(ctx, restoredToken)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	oauthClient := p.OAuth2.Client(ctx, token)
	resp, err := oauthClient.Get(p.OAuth2.Endpoint.AuthURL + "/userinfo") // todo parse root path
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
