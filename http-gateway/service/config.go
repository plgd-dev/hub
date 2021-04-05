package service

import (
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

// OAuthClientConfig represents oauth configuration for user interface exposed via getOAuthConfiguration handler
type OAuthClientConfig struct {
	Domain             string `json:"domain" yaml:"domain" envconfig:"DOMAIN"`
	ClientID           string `json:"clientID" yaml:"clientID" envconfig:"CLIENT_ID"`
	Audience           string `json:"audience" yaml:"audience" envconfig:"AUDIENCE"`
	Scope              string `json:"scope" yaml:"scope" envconfig:"SCOPE"`
	HTTPGatewayAddress string  `json:"httpGatewayAddress" yaml:"httpGatewayAddress" envconfig:"HTTP_GATEWAY_ADDRESS"`
}

// UIConfig represents user interface configuration
type UIConfig struct {
	Enabled     bool              `json:"enabled" yaml:"enabled" envconfig:"ENABLED"`
	Directory   string            `json:"directory" yaml:"directory" envconfig:"DIRECTORY"`
	OAuthClient OAuthClientConfig `json:"oauthClient" yaml:"oauthClient" envconfig:"OAUTH"`
}

// Config represents application configuration
type Config struct {
	Log        log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service    APIsConfig	   `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	   ClientsConfig   `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
	UI         UIConfig        `yaml:"ui" json:"ui" envconfig:"UI"`
}

type APIsConfig struct {
	Http HttpConfig `yaml:"http" json:"http" envconfig:"HTTP"`
}

type HttpConfig struct {
	Addr         string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9086"`
	TLSConfig    server.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
	Capabilities Capabilities  `yaml:"capabilities" json:"capabilities" envconfig:"CAPABILITIES"`
}

type Capabilities struct {
	WebSocketReadLimit      int64         `yaml:"websocketReadLimit" json:"websocketReadLimit" envconfig:"WEBSOCKET_READ_LIMIT" default:"8192"`
	WebSocketReadTimeout    time.Duration `yaml:"websocketReadTimeout" json:"websocketReadTimeout" envconfig:"WEBSOCKET_READ_TIMEOUT" default:"4s"`
}

type ClientsConfig struct {
	Nats              nats.Config      `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider     OAuthProvider    `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	CertificateAuthority CertificateAuthorityConfig `yaml:"certificateAuthority" json:"certificateAuthority" envconfig:"CERTIFICATE_AUTHORITY"`
	ResourceDirectory ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory" envconfig:"RESOURCE_DIRECTORY"`
	ResourceAggregate ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate" envconfig:"RESOURCE_AGGREGATE"`
	GoRoutinePoolSize int               `yaml:"goRoutinePoolSize" json:"goRoutinePoolSize" envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
}

type OAuthProvider struct {
	JwksURL   string        `yaml:"jwksUrl" json:"jwksUrl" envconfig:"JWKS_URL"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceDirectoryConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9082"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceAggregateConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9083"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type CertificateAuthorityConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9087"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

func (c Config) checkForDefaults() Config {
	if c.Service.Http.Capabilities.WebSocketReadLimit == 0 {
		c.Service.Http.Capabilities.WebSocketReadLimit = 8192
	}
	if c.Service.Http.Capabilities.WebSocketReadTimeout == 0 {
		c.Service.Http.Capabilities.WebSocketReadTimeout = time.Second * 4
	}
	if c.UI.Directory == "" {
		c.UI.Directory = "/usr/local/var/www"
	}

	return c
}

func (c Config) String() string {
	return config.ToString(c)
}
