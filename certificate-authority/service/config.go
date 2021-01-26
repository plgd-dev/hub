package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
	"time"
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
	OAuthProvider OAuthProvider   `yaml:"oauthProvider" json:"oauthProvider"`
	SignerConfig  SignerConfig    `yaml:"signer" json:"signer"`
}

type OAuthProvider struct {
	JwksURL        string         `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthTLSConfig client.Config  `yaml:"tls" json:"tls"`
}

type SignerConfig struct {
	Certificate   string           `yaml:"certificate" json:"certificate"`
	PrivateKey    string           `yaml:"privateKey" json:"privateKey"`
	ValidFrom     ValidFromDecoder `yaml:"-" json:"validFrom" default:"now"`
	ValidDuration time.Duration    `yaml:"-" json:"validDuration" default:"87600h"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}