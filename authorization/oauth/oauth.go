package oauth

import "golang.org/x/oauth2"

type AuthStyle string

const (
	// AuthStyleAutoDetect means to auto-detect which authentication
	// style the provider wants by trying both ways and caching
	// the successful way for the future.
	AuthStyleAutoDetect AuthStyle = "AutoDetect"

	// AuthStyleInParams sends the "client_id" and "client_secret"
	// in the POST body as application/x-www-form-urlencoded parameters.
	AuthStyleInParams AuthStyle = "InParams"

	// AuthStyleInHeader sends the client_id and client_password
	// using HTTP Basic Authorization. This is an optional style
	// described in the OAuth2 RFC 6749 section 2.3.1.
	AuthStyleInHeader AuthStyle = "InHeader"
)

func (a AuthStyle) ToOAuth2() oauth2.AuthStyle {
	switch a {
	case AuthStyleInHeader:
		return oauth2.AuthStyleInHeader
	case AuthStyleInParams:
		return oauth2.AuthStyleInParams
	}
	return oauth2.AuthStyleAutoDetect
}

type Endpoint struct {
	AuthURL   string    `json:"AuthUrl" envconfig:"AUTH_URL" env:"AUTH_URL" long:"auth_url"`
	TokenURL  string    `json:"TokenUrl" envconfig:"TOKEN_URL" env:"TOKEN_URL" long:"token_url"`
	AuthStyle AuthStyle `envconfig:"AUTH_STYLE" env:"AUTH_STYLE" long:"auth_style"`
}

type Config struct {
	ClientID     string   `json:"ClientId" envconfig:"CLIENT_ID" env:"CLIENT_ID" long:"client_id"`
	ClientSecret string   `json:"ClientSecret" envconfig:"CLIENT_SECRET" env:"CLIENT_SECRET" long:"client_secret"`
	Scopes       []string `json:"Scopes" envconfig:"SCOPES" env:"SCOPES" long:"scopes"`
	Endpoint     Endpoint `json:"Endpoint" envconfig:"ENDPOINT" env:"ENDPOINT" long:"endpoint"`
	Audience     string   `json:"Audience" envconfig:"AUDIENCE" env:"AUDIENCE" long:"audience"`
	RedirectURL  string   `json:"RedirectUrl" envconfig:"REDIRECT_URL" env:"REDIRECT_URL" long:"redirect_url"`
	AccessType   string   `json:"AccessType" envconfig:"ACCESS_TYPE" env:"ACCESS_TYPE" long:"access_type"`
	ResponseType string   `json:"ResponseType" envconfig:"RESPONSE_TYPE" env:"RESPONSE_TYPE" long:"response_type"`
	ResponseMode string   `json:"ResponseMode" envconfig:"RESPONSE_MODE" env:"RESPONSE_MODE" long:"response_mode"`
}

func (c Config) ToOAuth2() oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Scopes:       c.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   c.Endpoint.AuthURL,
			TokenURL:  c.Endpoint.TokenURL,
			AuthStyle: c.Endpoint.AuthStyle.ToOAuth2(),
		},
	}
}

func (c Config) AuthCodeURL(csrfToken string) string {
	aud := c.Audience
	accessType := c.AccessType
	responseType := c.ResponseType
	responseMode := c.ResponseMode

	opts := make([]oauth2.AuthCodeOption, 0, 2)

	if accessType != "" {
		opts = append(opts, oauth2.SetAuthURLParam("access_type", accessType))
	}
	if aud != "" {
		opts = append(opts, oauth2.SetAuthURLParam("audience", aud))
	}
	if responseType != "" {
		opts = append(opts, oauth2.SetAuthURLParam("response_type", responseType))
	}
	if responseMode != "" {
		opts = append(opts, oauth2.SetAuthURLParam("response_mode", responseMode))
	}
	auth := c.ToOAuth2()
	return (&auth).AuthCodeURL(csrfToken, opts...)
}
