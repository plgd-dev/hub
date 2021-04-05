package service

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/config"
	"time"

	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-gateway/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

//Config represent application configuration
type Config struct {
	Log         log.Config       `yaml:"log" json:"log" envconfig:"LOG"`
	Service     APIsConfig	     `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	    ClientsConfig    `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
	Database    MogoDBConfig     `yaml:"database" json:"database" envconfig:"DATABASE"`
}

type APIsConfig struct {
	Http            HttpConfig      `yaml:"http" json:"http" envconfig:"HTTP"`
	Capabilities    Capabilities    `yaml:"capabilities,omitempty" json:"capabilities" envconfig:"CAPABILITIES"`
}

type HttpConfig struct {
	Addr         string           `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9088"`
	TLSConfig    server.Config    `yaml:"tls" json:"tls" envconfig:"TLS"`
	FQDN         string           `yaml:"fqdn" json:"fqdn" envconfig:"FQDN" default:"cloud2cloud.pluggedin.cloud"`
}

type Capabilities struct {
	ReconnectInterval    time.Duration `yaml:"reconnectInterval" json:"reconnectInterval" envconfig:"RECONNECT_INTERVAL" default:"10s"`
	EmitEventTimeout     time.Duration `yaml:"emitEventTimeout" json:"emitEventTimeout" envconfig:"EMIT_EVENT_TIMEOUT" default:"5s"`
	GoRoutinePoolSize    int           `yaml:"goRoutinePoolSize" json:"goRoutinePoolSize" envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
}

type ClientsConfig struct {
	Nats             	 nats.Config   `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider        OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	ResourceDirectory    ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory" envconfig:"RESOURCE_DIRECTORY"`
	ResourceAggregate    ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate" envconfig:"RESOURCE_AGGREGATE"`
}

type MogoDBConfig struct {
	MongoDB    storeMongodb.Config    `yaml:"mongoDB" json:"mongoDB" envconfig:"MONGODB"`
}

type OAuthProvider struct {
	JwksURL    string         `yaml:"jwksUrl" json:"jwksUrl" envconfig:"JWKS_URL"`
	OwnerClaim string         `yaml:"ownerClaim" json:"ownerClaim" envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	OAuth      manager.Config `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig  client.Config  `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceDirectoryConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9082"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceAggregateConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9083"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}


//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
