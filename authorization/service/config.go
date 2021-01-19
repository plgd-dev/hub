package service

import (
	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/server"
	"github.com/plgd-dev/kit/security/certManager/client"

)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log                     log.Config               `yaml:"log" json:"log"`
	Service	                APIsConfig               `yaml:"apis" json:"apis"`
	Clients	                OAuthClientsConfig       `yaml:"oAuthClients" json:"oAuthClients"`
	Database                MogoDBConfig             `yaml:"database" json:"database"`
}

type APIsConfig struct {
	GrpcServer              GrpcConfig               `yaml:"grpc" json:"grpc"`
	HttpServer              HttpConfig               `yaml:"http" json:"http"`
}

type GrpcConfig struct {
	GrpcAddr                string                   `yaml:"address" json:"address" default:"0.0.0.0:9081"`
	GrpcTLSConfig           server.Config            `yaml:"tls" json:"tls"`
}

type HttpConfig struct {
	HttpAddr                string                   `yaml:"address" json:"address" default:"0.0.0.0:9085"`
	HttpTLSConfig           server.Config            `yaml:"tls" json:"tls"`
}

type OAuthClientsConfig struct {
	Device                  provider.Config          `yaml:"device" json:"device"`
	SDK                     SDKOAuthConfig           `yaml:"client" json:"client"`
}

type SDKOAuthConfig struct {
	OAuth                   oauth.Config             `yaml:"oauth" json:"oAuth"`
	OAuthTLSConfig          client.Config            `yaml:"tls" json:"tls"`
}

type MogoDBConfig struct {
	MongoDB                 mongodb.Config           `yaml:"mongoDB" json:"mongoDB"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
