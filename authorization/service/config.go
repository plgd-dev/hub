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
	Log         log.Config         `yaml:"log" json:"log" envconfig:"LOG" default:"true"`
	Service     APIsConfig         `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients     OAuthClientsConfig `yaml:"oauthClients" json:"oauthClients" envconfig:"CLIENTS"`
	Database    MogoDBConfig       `yaml:"database" json:"database" envconfig:"DATABASE"`
}

type APIsConfig struct {
	Grpc GrpcConfig `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
	Http HttpConfig `yaml:"http" json:"http" envconfig:"HTTP"`
}

type GrpcConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9081"`
	TLSConfig server.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type HttpConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9085"`
	TLSConfig server.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type OAuthClientsConfig struct {
	Device    provider.Config    `yaml:"device" json:"device" envconfig:"DEVICE"`
	SDK       SDKOAuthConfig     `yaml:"client" json:"client" envconfig:"SDK"`
}

type SDKOAuthConfig struct {
	OAuth     oauth.Config  `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type MogoDBConfig struct {
	MongoDB    mongodb.Config    `yaml:"mongoDB" json:"mongoDB" envconfig:"MONGODB"`
}

func (c Config) CheckForDefaults() Config {
	if c.Clients.Device.OAuth.AccessType == "" {
		c.Clients.Device.OAuth.AccessType = "offline"
	}
	if c.Clients.Device.OAuth.ResponseType == "" {
		c.Clients.Device.OAuth.ResponseType = "code"
	}
	if c.Clients.Device.OAuth.ResponseMode == "" {
		c.Clients.Device.OAuth.ResponseMode = "query"
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
