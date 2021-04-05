package service

import (
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"time"
)

//Config represent application configuration
type Config struct {
	Log         log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service	    APIsConfig      `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	    ClientsConfig   `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
	Database    Database        `yaml:"database" json:"database" envconfig:"DATABASE"`
}

type APIsConfig struct {
	Grpc GrpcConfig `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
}

type GrpcConfig struct {
	Addr          string             `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9083"`
	TLSConfig     server.Config      `yaml:"tls" json:"tls" envconfig:"TLS"`
	Capabilities  CapabilitiesConfig `yaml:"capabilities" json:"capabilities" envconfig:"CAPABILITIES"`
}

type CapabilitiesConfig struct {
	SnapshotThreshold               int            `yaml:"snapshotThreshold" json:"snapshotThreshold" envconfig:"SNAPSHOT_THRESHOLD" default:"16"`
	ConcurrencyExceptionMaxRetry    int            `yaml:"occMaxRetry" json:"occMaxRetry" envconfig:"OCC_MAX_RETRY" default:"8"`
	UserDevicesManagerTickFrequency time.Duration  `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `yaml:"userMgmtExpiration" json:"userMgmtExpiration" envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
	NumParallelRequest              int            `yaml:"maxParallelRequest" json:"maxParallelRequest" envconfig:"MAX_PARALLEL_REQUEST" default:"8"`
}

type Database struct {
	MongoDB    mongodb.Config    `yaml:"mongoDB" json:"mongoDB" envconfig:"MONGODB"`
}

type ClientsConfig struct {
	Nats          nats.Config   `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider OAuthProvider `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	Authorization AuthServer    `yaml:"authorizationServer" json:"authorizationServer" envconfig:"AUTHORIZATION"`
}

type OAuthProvider struct {
	JwksURL    string         `yaml:"jwksUrl" json:"jwksUrl" envconfig:"JWKS_URL"`
	OwnerClaim string         `yaml:"ownerClaim" json:"ownerClaim" envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	OAuth      manager.Config `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig  client.Config  `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type AuthServer struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9100"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

