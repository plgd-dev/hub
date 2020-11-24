package service

import (
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager"
)

// Config represent application configuration
type Config struct {
	Address                  string             `envconfig:"ADDRESS"`
	Listen                   certManager.Config `envconfig:"LISTEN"`
	Dial                     certManager.Config `envconfig:"DIAL"`
	JwksURL                  string             `envconfig:"JWKS_URL"`
	ResourceDirectoryAddr    string             `envconfig:"RESOURCE_DIRECTORY_ADDRESS"`
	CertificateAuthorityAddr string             `envconfig:"CERTIFICATE_AUTHORITY_ADDRESS"`
	WebSocketReadLimit       int64              `envconfig:"WEBSOCKET_READ_LIMIT"`
	WebSocketReadTimeout     time.Duration      `envconfig:"WEBSOCKET_READ_TIMEOUT"`
}

func (c Config) checkForDefaults() Config {
	if c.WebSocketReadLimit == 0 {
		c.WebSocketReadLimit = 8192
	}
	if c.WebSocketReadTimeout == 0 {
		c.WebSocketReadTimeout = time.Second * 4
	}
	return c
}

func (c Config) String() string {
	return config.ToString(c)
}
