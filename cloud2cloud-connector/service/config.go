package service

import (
	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	oauthClient "github.com/plgd-dev/kit/security/oauth/service/client"
	"time"
)

//Config represent application configuration
type Config struct {
	Log              log.Config     `yaml:"log" json:"log"`
	Service          APIsConfig	    `yaml:"apis" json:"apis"`
	Clients			 ClientsConfig  `yaml:"clients" json:"clients"`
	Database         MogoDBConfig   `yaml:"database" json:"database"`
}

type APIsConfig struct {
	HttpConfig            HttpConfig          `yaml:"http" json:"http"`
	PullDevicesDisabled   bool                `yaml:"pullDevicesDisabled" json:"pullDevicesDisabled" default:"false"`
	PullDevicesInterval   time.Duration       `yaml:"pullDevicesInterval" json:"pullDevicesInterval" default:"5s"`
	ReconnectInterval     time.Duration       `yaml:"reconnectInterval" json:"reconnectInterval" default:"10s"`
	ResubscribeInterval   time.Duration       `yaml:"resubscribeInterval" json:"resubscribeInterval" default:"10s"`
	TaskProcessor         TaskProcessorConfig `yaml:"taskProcessor" json:"taskProcessor"`
}

type HttpConfig struct {
	HttpAddr          string           `yaml:"address" json:"address" default:"0.0.0.0:9089"`
	HttpTLSConfig     server.Config    `yaml:"tls" json:"tls"`
	OAuthCallback     string           `yaml:"callbackURL" json:"callbackURL"`
	EventsURL         string           `yaml:"eventURL" json:"eventURL"`
}

type ClientsConfig struct {
	OAuthProvider         OAuthProvider           `yaml:"oAuthProvider" json:"oAuthProvider"`
	Authorization         AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer"`
	ResourceDirectory     ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory"`
	ResourceAggregate     ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate"`
}

type MogoDBConfig struct {
	MongoDB    storeMongodb.Config    `yaml:"mongoDB" json:"mongoDB"`
}

type TaskProcessorConfig struct {
	CacheSize   int           `yaml:"cacheSize" json:"cacheSize" default:"2048"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout" default:"5s"`
	MaxParallel int64         `yaml:"maxParallel" json:"maxParallel" default:"128"`
	Delay       time.Duration `yaml:"delay" json:"delay"` // Used for CTT test with 10s.
}

type OAuthProvider struct {
	JwksURL               string              `yaml:"jwksUrl" json:"jwksUrl"`
	OAuthConfig           oauthClient.Config  `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig        client.Config       `yaml:"tls" json:"tls"`
}

type AuthorizationConfig struct {
	AuthServerAddr                  string              `yaml:"address" json:"address" default:"127.0.0.1:9081"`
	AuthServerTLSConfig             client.Config       `yaml:"tls" json:"tls"`
}

type ResourceDirectoryConfig struct {
	ResourceDirectoryAddr           string              `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig      client.Config       `yaml:"tls" json:"tls"`
}

type ResourceAggregateConfig struct {
	ResourceAggregateAddr            string             `yaml:"address" json:"address" default:"127.0.0.1:9083"`
	ResourceAggregateTLSConfig       client.Config      `yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
