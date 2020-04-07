package store

import "golang.org/x/oauth2"

type Endpoint struct {
	AuthUrl  string `json:"AuthUrl" envconfig:"AUTH_URL" required:"true"`
	TokenUrl string `json:"TokenUrl" envconfig:"TOKEN_URL" required:"true"`
}

type LinkedCloud struct {
	ID           string   `json:"ID"`
	Name         string   `json:"Name" envconfig:"NAME" required:"true"`
	ClientID     string   `json:"ClientId" envconfig:"CLIENT_ID" required:"true"`
	ClientSecret string   `json:"ClientSecret" envconfig:"CLIENT_SECRET" required:"true"`
	Scopes       []string `json:"Scopes" envconfig:"SCOPES" required:"true"`
	Endpoint     Endpoint `json:"Endpoint"`
	Audience     string   `json:"Audience" envconfig:"AUDIENCE"`
}

func (l LinkedCloud) ToOAuth2Config() oauth2.Config {
	return oauth2.Config{
		ClientID:     l.ClientID,
		ClientSecret: l.ClientSecret,
		Scopes:       l.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  l.Endpoint.AuthUrl,
			TokenURL: l.Endpoint.TokenUrl,
		},
	}
}
