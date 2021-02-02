package service

import (
	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log         log.Config         `yaml:"log" json:"log"`
	Service     APIsConfig         `yaml:"apis" json:"apis"`
	Clients     OAuthClientsConfig `yaml:"oauthClients" json:"oauthClients"`
	Database    MogoDBConfig       `yaml:"database" json:"database"`
}

type APIsConfig struct {
	GrpcServer    GrpcConfig   `yaml:"grpc" json:"grpc"`
	HttpServer    HttpConfig   `yaml:"http" json:"http"`
}

type GrpcConfig struct {
	GrpcAddr         string        `yaml:"address" json:"address" default:"0.0.0.0:9081"`
	GrpcTLSConfig    server.Config `yaml:"tls" json:"tls"`
}

type HttpConfig struct {
	HttpAddr         string        `yaml:"address" json:"address" default:"0.0.0.0:9085"`
	HttpTLSConfig    server.Config `yaml:"tls" json:"tls"`
}

type OAuthClientsConfig struct {
	Device    provider.Config    `yaml:"device" json:"device"`
	SDK       SDKOAuthConfig     `yaml:"client" json:"client"`
}

type SDKOAuthConfig struct {
	OAuth             oauth.Config     `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig    client.Config    `yaml:"tls" json:"tls"`
}

type MogoDBConfig struct {
	MongoDB    mongodb.Config    `yaml:"mongoDB" json:"mongoDB"`
}

func (c Config) CheckForDefaults() Config {
	if c.Clients.Device.OAuth2.AccessType == "" {
		c.Clients.Device.OAuth2.AccessType = "offline"
	}
	if c.Clients.Device.OAuth2.ResponseType == "" {
		c.Clients.Device.OAuth2.ResponseType = "code"
	}
	if c.Clients.Device.OAuth2.ResponseMode == "" {
		c.Clients.Device.OAuth2.ResponseMode = "query"
	}
	if c.Clients.SDK.OAuth.AccessType == "" {
		c.Clients.SDK.OAuth.AccessType = "online"
	}
	if c.Clients.SDK.OAuth.ResponseType == "" {
		c.Clients.SDK.OAuth.ResponseType = "token"
	}
	if c.Clients.SDK.OAuth.ResponseMode == "" {
		c.Clients.SDK.OAuth.ResponseMode = "query"
	}
	return c
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
