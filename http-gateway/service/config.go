package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certificateManager"
)

// Config represent application configuration
type Config struct {
	Address                  string                    `envconfig:"ADDRESS" default:"0.0.0.0:7000"`
	Listen                   certificateManager.Config `envconfig:"LISTEN"`
	Dial                     certificateManager.Config `envconfig:"DIAL"`
	JwksURL                  string                    `envconfig:"JWKS_URL"`
	ResourceDirectoryAddr    string                    `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	CertificateAuthorityAddr string                    `envconfig:"CERTIFICATE_AUTHORITY_ADDRESS"  default:""`
}

func (c Config) String() string {
	return config.ToString(c)
}
