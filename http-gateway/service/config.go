package service

import (
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager"
)

// OAuthClientConfig represents oauth configuration for user interface exposed via getOAuthConfiguration handler
type OAuthClientConfig struct {
	Domain   string `json:"domain" yaml:"domain"`
	ClientID string `json:"clientID" yaml:"clientID"`
	Audience string `json:"audience" yaml:"audience"`
	Scope    string `json:"scope" yaml:"scope"`
}

// UIConfig represents user interface configuration
type UIConfig struct {
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Directory   string            `json:"directory" yaml:"directory"`
	OAuthClient OAuthClientConfig `json:"oauthClient" yaml:"oauthClient"`
}

// Config represents application configuration
type Config struct {
	Address                  string
	Listen                   certManager.Config
	Dial                     certManager.Config
	JwksURL                  string
	ResourceDirectoryAddr    string
	CertificateAuthorityAddr string
	WebSocketReadLimit       int64
	WebSocketReadTimeout     time.Duration
	UI                       UIConfig
}

func (c Config) checkForDefaults() Config {
	if c.WebSocketReadLimit == 0 {
		c.WebSocketReadLimit = 8192
	}
	if c.WebSocketReadTimeout == 0 {
		c.WebSocketReadTimeout = time.Second * 4
	}
	if c.UI.Directory == "" {
		c.UI.Directory = "/usr/local/var/www"
	}

	return c
}

func (c Config) String() string {
	return config.ToString(c)
}
