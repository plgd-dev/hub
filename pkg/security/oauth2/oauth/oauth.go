package oauth

import (
	"fmt"

	"golang.org/x/oauth2"
)

type AuthStyle string

const (
	// AuthStyleAutoDetect means to auto-detect which authentication
	// style the provider wants by trying both ways and caching
	// the successful way for the future.
	AuthStyleAutoDetect AuthStyle = "autoDetect"

	// AuthStyleInParams sends the "client_id" and "client_secret"
	// in the POST body as application/x-www-form-urlencoded parameters.
	AuthStyleInParams AuthStyle = "inParams"

	// AuthStyleInHeader sends the client_id and client_password
	// using HTTP Basic Authorization. This is an optional style
	// described in the OAuth2 RFC 6749 section 2.3.1.
	AuthStyleInHeader AuthStyle = "inHeader"
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

//TODO cleanup settings AccessType, ResponseType, ResponseMode, AuthStyle - be careful it is used by c2c.
type Config struct {
	ClientID         string    `yaml:"clientID" json:"clientId"`
	ClientSecretFile string    `yaml:"clientSecretFile" json:"clientSecretFile"`
	Scopes           []string  `yaml:"scopes" json:"scopes"`
	AuthURL          string    `yaml:"-" json:"authUrl"`
	TokenURL         string    `yaml:"-" json:"tokenUrl"`
	AuthStyle        AuthStyle `yaml:"authStyle" json:"authStyle"`
	Audience         string    `yaml:"audience" json:"audience"`
	RedirectURL      string    `yaml:"redirectURL" json:"redirectUrl"`
	AccessType       string    `yaml:"accessType" json:"accessType"`
	ResponseType     string    `yaml:"responseType" json:"responseType"`
	ResponseMode     string    `yaml:"responseMode" json:"responseMode"`
	ClientSecret     string    `yaml:"-" json:"clientSecret"`
}

func (c *Config) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("clientID('%v')", c.ClientID)
	}
	if c.ClientSecretFile == "" {
		return fmt.Errorf("clientSecretFile('%v')", c.ClientSecretFile)
	}
	return nil
}

func (c Config) ToOAuth2(authURL, tokenURL, clientSecret string) oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  c.RedirectURL,
		Scopes:       c.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: c.AuthStyle.ToOAuth2(),
		},
	}
}

func (c Config) ToDefaultOAuth2() oauth2.Config {
	return c.ToOAuth2(c.AuthURL, c.TokenURL, c.ClientSecret)
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
	auth := c.ToOAuth2(c.AuthURL, c.TokenURL, c.ClientSecret)
	return (&auth).AuthCodeURL(csrfToken, opts...)
}
