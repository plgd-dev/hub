package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
)

// Config represent application configuration
type Config struct {
	Log        log.Config      `yaml:"log" json:"log"`
	Service    APIsConfig	   `yaml:"apis" json:"apis"`
	Clients	   ClientsConfig   `yaml:"clients" json:"clients"`
}

type APIsConfig struct {
	GrpcConfig    kitNetGrpc.Config    `yaml:"grpc" json:"grpc"`
}

type ClientsConfig struct {
	OAuthProvider OAuthProvider    `yaml:"oauthProvider" json:"oauthProvider"`
	RDConfig      RDConfig         `yaml:"resourceDirectory" json:"resourceDirectory"`
}

type OAuthProvider struct {
	JwksURL        string         `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthTLSConfig client.Config  `yaml:"tls" json:"tls"`
}

type RDConfig struct {
	ResourceDirectoryAddr      string        `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig client.Config `yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}