package service

import (
	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"time"
)

//Config represent application configuration
type Config struct {
	Log              log.Config     `yaml:"log" json:"log" envconfig:"LOG"`
	Service          APIsConfig	    `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients			 ClientsConfig  `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
	Database         MogoDBConfig   `yaml:"database" json:"database" envconfig:"DATABASE"`
}

type APIsConfig struct {
	Http             HttpConfig          `yaml:"http" json:"http" envconfig:"HTTP"`
	Capabilities     Capabilities        `yaml:"capabilities,omitempty" json:"capabilities" envconfig:"CAPABILITIES"`
	TaskProcessor    TaskProcessorConfig `yaml:"taskProcessor,omitempty" json:"taskProcessor" envconfig:"TASK_PROCESSOR"`
}

type HttpConfig struct {
	Addr              string           `yaml:"address" json:"address" default:"0.0.0.0:9089" envconfig:"ADDRESS"`
	TLSConfig         server.Config    `yaml:"tls" json:"tls" envconfig:"TLS"`
	OAuthCallback     string           `yaml:"callbackURL" json:"callbackURL" envconfig:"CALLBACK_URL"`
	EventsURL         string           `yaml:"eventsURL" json:"eventsURL" envconfig:"EVENTS_URL"`
}

type Capabilities struct {
	PullDevicesDisabled   bool             `yaml:"pullDevicesDisabled,omitempty" json:"pullDevicesDisabled" envconfig:"PULL_DEVICES_DISABLED" default:"false"`
	PullDevicesInterval   time.Duration    `yaml:"pullDevicesInterval,omitempty" json:"pullDevicesInterval" envconfig:"PULL_DEVICES_INTERVAL" default:"5s"`
	ReconnectInterval     time.Duration    `yaml:"reconnectInterval,omitempty" json:"reconnectInterval" envconfig:"RECONNECT_INTERVAL" default:"10s"`
	ResubscribeInterval   time.Duration    `yaml:"resubscribeInterval,omitempty" json:"resubscribeInterval" envconfig:"RESUBSCRIBE_INTERVAL" default:"10s"`
}

type ClientsConfig struct {
	OAuthProvider         OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	Authorization         AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer" envconfig:"AUTHORIZATION"`
	ResourceDirectory     ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory" envconfig:"RESOURCE_DIRECTORY"`
	ResourceAggregate     ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate" envconfig:"RESOURCE_AGGREGATE"`
}

type MogoDBConfig struct {
	MongoDB    storeMongodb.Config    `yaml:"mongoDB" json:"mongoDB" envconfig:"MONGODB"`
}

type TaskProcessorConfig struct {
	CacheSize   int           `yaml:"cacheSize,omitempty" json:"cacheSize" envconfig:"CACHE_SIZE" default:"2048"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout" envconfig:"TIMEOUT" default:"5s"`
	MaxParallel int64         `yaml:"maxParallel,omitempty" json:"maxParallel" envconfig:"MAX_PARALLEL" default:"128"`
	Delay       time.Duration `yaml:"delay,omitempty" json:"delay" envconfig:"DELAY"` // Used for CTT test with 10s.
}


type OAuthProvider struct {
	JwksURL    string         `yaml:"jwksUrl" json:"jwksUrl" envconfig:"JWKS_URL"`
	OwnerClaim string         `yaml:"ownerClaim" json:"ownerClaim" envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	OAuth      manager.Config `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig  client.Config  `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type AuthorizationConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9081"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceDirectoryConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9082"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceAggregateConfig struct {
	Addr      string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9083"`
	TLSConfig client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
