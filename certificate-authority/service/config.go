package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"time"
)

// Config represent application configuration
type Config struct {
	Log        log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service    APIsConfig	   `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	   ClientsConfig   `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
}

type APIsConfig struct {
	Grpc GrpcConfig `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
}

type GrpcConfig struct {
	Addr          string           `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9087"`
	TLSConfig     server.Config    `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ClientsConfig struct {
	OAuthProvider OAuthProvider `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	Signer        SignerConfig  `yaml:"signer" json:"signer" envconfig:"SIGNER"`
}

type OAuthProvider struct {
	JwksURL   string        `yaml:"jwksUrl" json:"jwksUrl" envconfig:"JWKS_URL"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type SignerConfig struct {
	Certificate   string           `yaml:"certificate" json:"certificate" envconfig:"CERTIFICATE"`
	PrivateKey    string           `yaml:"privateKey" json:"privateKey" envconfig:"PRIVATE_KEY"`
	ValidFrom     ValidFromDecoder `yaml:"-" json:"validFrom" default:"now"`
	ValidDuration time.Duration    `yaml:"-" json:"validDuration" default:"87600h"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}