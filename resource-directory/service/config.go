package service

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
)

type Config struct {
	Log         log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service	    APIsConfig      `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	    ClientsConfig   `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
	Database    Database        `yaml:"database" json:"database" envconfig:"DATABASE"`
}

// Config represent application configuration
type APIsConfig struct {
	Grpc    GrpcConfig     `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
}

type GrpcConfig struct {
	Addr          string                 `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9100"`
	TLSConfig     server.Config          `yaml:"tls" json:"tls" envconfig:"TLS"`
	FQDN          string                 `yaml:"fqdn" json:"fqdn" envconfig:"FQDN" default:"grpcgw.ocf.cloud"`
	Capabilities  CapabilitiesConfig     `yaml:"capabilities" json:"capabilities" envconfig:"CAPABILITIES"`
}

type CapabilitiesConfig struct {
	TimeoutForRequests        time.Duration  `yaml:"timeout" json:"timeout" envconfig:"TIMEOUT" default:"10s"`
	ProjectionCacheExpiration time.Duration  `yaml:"cacheExpiration" json:"cacheExpiration" envconfig:"CACHE_EXPIRATION" default:"1m"`
	GoRoutinePoolSize         int            `yaml:"goRoutinePoolSize" json:"goRoutinePoolSize" envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	UserDevicesManagerTickFrequency time.Duration  `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `yaml:"userMgmtExpiration" json:"userMgmtExpiration" envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
}

type Database struct {
	MongoDB    mongodb.Config     `yaml:"mongoDB" json:"mongoDB" envconfig:"MONGODB"`
}

type ClientsConfig struct {
	Nats                      nats.Config             `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider             OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	Authorization             AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer" envconfig:"AUTHORIZATION"`
	ResourceAggregate         ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate" envconfig:"RESOURCE_AGGREGATE"`
	ClientConfiguration       CloudConfig             `yaml:"clientConfiguration" envconfig:"CONFIG"`
}

type OAuthProvider struct {
	JwksURL    string         `yaml:"jwksUrl" json:"jwksURL" envconfig:"JWKS_URL"`
	OwnerClaim string         `yaml:"ownerClaim" json:"ownerClaim" envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	OAuth      manager.Config `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig  client.Config  `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type AuthorizationConfig struct {
	Addr            string              `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9081"`
	TLSConfig       client.Config       `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceAggregateConfig struct {
	Addr            string             `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9083"`
	TLSConfig       client.Config      `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type CloudConfig struct {
	CloudCAPool     string             `yaml:"cloudCAPool" json:"cloudCAPool" envconfig:"CLOUD_CA_POOL" description:"file path to the root certificate in PEM format"`
	pb.ClientConfigurationResponse     `yaml:",inline"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
