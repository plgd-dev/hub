package service

import (
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager"
)

// Config represent application configuration
type Config struct {
	Address                  string
	Listen                   certManager.Config
	Dial                     certManager.Config
	JwksURL                  string
	ResourceDirectoryAddr    string
	CertificateAuthorityAddr string
	WebSocketReadLimit       int64
	WebSocketReadTimeout     time.Duration
	UIEnabled                bool
	UIDirectory              string
}

func (c Config) checkForDefaults() Config {
	if c.WebSocketReadLimit == 0 {
		c.WebSocketReadLimit = 8192
	}
	if c.WebSocketReadTimeout == 0 {
		c.WebSocketReadTimeout = time.Second * 4
	}
	if c.UIDirectory == "" {
		c.UIDirectory = "/usr/local/var/www"
	}

	return c
}

func (c Config) String() string {
	return config.ToString(c)
}
