package service

import (
	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log 					log.Config				`yaml:"log" json:"log"`
	Service					ServiceConfig 			`yaml:"apis" json:"apis"`
	Clients			 		ClientsConfig  			`yaml:"clients" json:"clients"`

}

type ServiceConfig struct {
	GrpcServer				GrpcConfig				`yaml:"grpc" json:"grpc"`
	HttpServer 				HttpConfig				`yaml:"http" json:"http"`
}

type GrpcConfig struct {
	GrpcAddr     			string             		`yaml:"address" json:"address" default:"0.0.0.0:9081"`
	GrpcTLSConfig			certManager.Config 		`yaml:"tls" json:"tls"`
}

type HttpConfig struct {
	HttpAddr 				string             		`yaml:"address" json:"address" default:"0.0.0.0:9085"`
	HttpTLSConfig			certManager.Config 		`yaml:"tls" json:"tls"`
}

type ClientsConfig struct {
	DeviceConfig 			provider.Config 		`yaml:"device-oauth" json:"device-oauth"`
	SDKConfig				SDKOAuthConfig			`yaml:"sdk-oauth" json:"sdk-oauth"`
	MogoDBConfig			MogoDBConfig			`yaml:"mongo" json:"mongo"`
}

type SDKOAuthConfig struct {
	OAuth    				oauth.Config    		`yaml:"oauth" json:"oauth"`
	OAuthTLSConfig			certManager.Config 		`yaml:"tls" json:"tls"`
}

type MogoDBConfig struct {
	MongoDB  				mongodb.Config     		`yaml:"mongodb" json:"mongodb"`
	MongoDBTLSConfig		certManager.Config 		`yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
