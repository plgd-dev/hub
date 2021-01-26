package service

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	client2 "github.com/plgd-dev/kit/security/oauth/service/client"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

type Config struct {
	Log         log.Config      `yaml:"log" json:"log"`
	Service	    APIsConfig      `yaml:"apis" json:"apis"`
	Clients	    ClientsConfig   `yaml:"clients" json:"clients"`
	Database    Database        `yaml:"database" json:"database"`
}

// Config represent application configuration
type APIsConfig struct {
	RD    GrpcConfig     `yaml:"grpc" json:"grpc"`
}

type GrpcConfig struct {
	GrpcAddr          string                 `yaml:"address" json:"address" default:"0.0.0.0:9100"`
	GrpcTLSConfig     server.Config          `yaml:"tls" json:"tls"`
	FQDN              string                 `yaml:"fqdn" json:"fqdn" default:"grpcgw.ocf.cloud"`
	Capabilities	  CapabilitiesConfig     `yaml:"capabilities" json:"capabilities"`
}

type CapabilitiesConfig struct {
	TimeoutForRequests        time.Duration  `yaml:"timeout" json:"timeout" default:"10s"`
	ProjectionCacheExpiration time.Duration  `yaml:"cacheExpiration" json:"cacheExpiration" default:"1m"`
	GoRoutinePoolSize               int            `yaml:"goRoutinePoolSize" json:"goRoutinePoolSize" default:"16"`
	UserDevicesManagerTickFrequency time.Duration  `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `yaml:"userMgmtExpiration" json:"userMgmtExpiration" default:"1m"`
}

type Database struct {
	MongoDB    mongodb.Config     `yaml:"mongoDB" json:"mongoDB"`
}

type ClientsConfig struct {
	Nats                      nats.Config             `yaml:"nats" json:"nats"`
	OAuthProvider             OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider"`
	Authorization             AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer"`
	ResourceAggregate         ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate"`
	ClientConfiguration       CloudConfig             `yaml:"clientConfiguration"`
}

type OAuthProvider struct {
	JwksURL        string         `yaml:"jwksUrl" json:"jwksURL"`
	OAuthConfig    client2.Config `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig client.Config  `yaml:"tls" json:"tls"`
}

type AuthorizationConfig struct {
	Addr            string              `yaml:"address" json:"address" default:"127.0.0.1:9081"`
	TLSConfig       client.Config       `yaml:"tls" json:"tls"`
}

type ResourceAggregateConfig struct {
	Addr            string             `yaml:"address" json:"address" default:"127.0.0.1:9083"`
	TLSConfig       client.Config      `yaml:"tls" json:"tls"`
}

type CloudConfig struct {
	CloudCAPool     string             `yaml:"cloudCAPool" json:"cloudCAPool" description:"file path to the root certificate in PEM format"`
	pb.ClientConfigurationResponse     `yaml:"cloudConfiguration" json:"cloudConfiguration"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
