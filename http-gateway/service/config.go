package service

import (
	"time"
	"github.com/plgd-dev/kit/log"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

// Config represent application configuration
type Config struct {
	Log        log.Config      `yaml:"log" json:"log"`
	Service    APIsConfig	   `yaml:"apis" json:"apis"`
	Clients	   ClientsConfig   `yaml:"clients" json:"clients"`
}

type APIsConfig struct {
	HttpConfig    HttpConfig    `yaml:"http" json:"http"`
	Capabilities  Capabilities  `yaml:"capabilities" json:"capabilities"`
}

type HttpConfig struct {
	HttpAddr          string           `yaml:"address" json:"address" default:"0.0.0.0:9086"`
	HttpTLSConfig     server.Config    `yaml:"tls" json:"tls"`
}

type Capabilities struct {
	WebSocketReadLimit       int64              `yaml:"websocketReadLimit" json:"websocketReadLimit" default:"8192"`
	WebSocketReadTimeout     time.Duration      `yaml:"websocketReadTimeout" json:"websocketReadTimeout" default:"4s"`
}

type ClientsConfig struct {
	OAuthProvider OAuthProvider    `yaml:"oAuthProvider" json:"oAuthProvider"`
	RDConfig      RDConfig         `yaml:"resourceDirectory" json:"resourceDirectory"`
	CAConfig      CAConfig         `yaml:"certificateAuthority" json:"certificateAuthority"`
}

type OAuthProvider struct {
	JwksURL        string         `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthTLSConfig client.Config  `yaml:"tls" json:"tls"`
}

type RDConfig struct {
	ResourceDirectoryAddr      string        `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig client.Config `yaml:"tls" json:"tls"`
}

type CAConfig struct {
	CertificateAuthorityAddr      string        `yaml:"address" json:"address" default:"127.0.0.1:9087"`
	CertificateAuthorityTLSConfig client.Config `yaml:"tls" json:"tls"`
}

func (c Config) checkForDefaults() Config {
	if c.Service.Capabilities.WebSocketReadLimit == 0 {
		c.Service.Capabilities.WebSocketReadLimit = 8192
	}
	if c.Service.Capabilities.WebSocketReadTimeout == 0 {
		c.Service.Capabilities.WebSocketReadTimeout = time.Second * 4
	}
	return c
}

func (c Config) String() string {
	return config.ToString(c)
}
