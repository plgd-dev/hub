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
)

//Config represent application configuration
type Config struct {
	Log         log.Config      `yaml:"log" json:"log"`
	Service	    APIsConfig      `yaml:"apis" json:"apis"`
	Clients	    ClientsConfig   `yaml:"clients" json:"clients"`
	Database    Database        `yaml:"database" json:"database"`
}

type APIsConfig struct {
	RA          GrpcServer       `yaml:"grpc" json:"grpc"`
}

type GrpcServer struct {
	GrpcAddr          string                  `yaml:"address" json:"address" default:"0.0.0.0:9100"`
	GrpcTLSConfig     server.Config           `yaml:"tls" json:"tls"`
	Capabilities	  CapabilitiesConfig      `yaml:"capabilities" json:"capabilities"`
}

type CapabilitiesConfig struct {
	SnapshotThreshold               int            `yaml:"snapshotThreshold" json:"snapshotThreshold" default:"16"`
	ConcurrencyExceptionMaxRetry    int            `yaml:"occMaxRetry" json:"occMaxRetry" default:"8"`
	UserDevicesManagerTickFrequency time.Duration  `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `yaml:"userMgmtExpiration" json:"userMgmtExpiration" default:"1m"`
	NumParallelRequest              int            `yaml:"maxParallelRequest" json:"maxParallelRequest" default:"8"`
}

type Database struct {
	MongoDB            mongodb.Config     `yaml:"mongoDB" json:"mongoDB"`
}

type ClientsConfig struct {
	Nats               nats.Config        `yaml:"nats" json:"nats"`
	OAuthProvider      OAuthProvider      `yaml:"oAuthProvider" json:"oAuthProvider"`
	AuthServer         AuthServer         `yaml:"authorizationServer" json:"authorizationServer"`
}

type OAuthProvider struct {
	JwksURL        string         `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthConfig    client2.Config `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig client.Config  `yaml:"tls" json:"tls"`
}

type AuthServer struct {
	AuthServerAddr     string         `yaml:"address" json:"address" default:"127.0.0.1:9100"`
	AuthTLSConfig      client.Config  `yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

