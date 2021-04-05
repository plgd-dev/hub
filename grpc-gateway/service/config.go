package service

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
)

// Config represent application configuration
type Config struct {
	Log        log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service    APIsConfig	   `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	   ClientsConfig   `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
}

type APIsConfig struct {
	GrpcConfig        kitNetGrpc.Config `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
}

type ClientsConfig struct {
	Nats              nats.Config      `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider     OAuthProvider    `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
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

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}