package service

import (
	"github.com/plgd-dev/kit/config"
	"time"

	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-gateway/store/mongodb"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	oauthClient "github.com/plgd-dev/kit/security/oauth/service/client"
)

//Config represent application configuration
type Config struct {
	Log              log.Config     `yaml:"log" json:"log"`
	Service          APIsConfig	    `yaml:"apis" json:"apis"`
	Clients			 ClientsConfig  `yaml:"clients" json:"clients"`
	Database         MogoDBConfig   `yaml:"database" json:"database"`
}

type APIsConfig struct {
	Http            HttpConfig      `yaml:"http" json:"http"`
	Capabilities    Capabilities    `yaml:"capabilities,omitempty" json:"capabilities"`
}

type HttpConfig struct {
	Addr          string           `yaml:"address" json:"address" default:"0.0.0.0:9088"`
	TLSConfig     server.Config    `yaml:"tls" json:"tls"`
	FQDN          string           `yaml:"fqdn" json:"fqdn" default:"cloud2cloud.pluggedin.cloud"`
}

type Capabilities struct {
	ReconnectInterval     time.Duration `yaml:"reconnectInterval" json:"reconnectInterval" default:"10s"`
	EmitEventTimeout      time.Duration `yaml:"emitEventTimeout" json:"emitEventTimeout" default:"5s"`
}

type ClientsConfig struct {
	OAuthProvider         OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider"`
	ResourceDirectory     ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory"`
}

type MogoDBConfig struct {
	MongoDB    storeMongodb.Config    `yaml:"mongoDB" json:"mongoDB"`
}

type OAuthProvider struct {
	JwksURL               string              `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthConfig           oauthClient.Config  `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig        client.Config       `yaml:"tls" json:"tls"`
}

type ResourceDirectoryConfig struct {
	ResourceDirectoryAddr           string              `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig      client.Config       `yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
