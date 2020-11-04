package service

import (
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/oauth/manager"
)

type TaskProcessorConfig struct {
	CacheSize   int           `envconfig:"CACHE_SIZE" default:"2048"`
	Timeout     time.Duration `envconfig:"TIMEOUT" default:"5s"`
	MaxParallel int64         `envconfig:"MAX_PARALLEL" default:"128"`
	Delay       time.Duration `envconfig:"DELAY"` // Used for CTT test with 10s.
}

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr        string              `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	ResourceAggregateAddr string              `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	ResourceDirectoryAddr string              `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	OAuthCallback         string              `envconfig:"OAUTH_CALLBACK"`
	EventsURL             string              `envconfig:"EVENTS_URL"`
	PullDevicesDisabled   bool                `envconfig:"PULL_DEVICES_DISABLED" default:"false"`
	PullDevicesInterval   time.Duration       `envconfig:"PULL_DEVICES_INTERVAL" default:"5s"`
	TaskProcessor         TaskProcessorConfig `envconfig:"TASK_PROCESSOR"`
	ReconnectInterval     time.Duration       `envconfig:"RECONNECT_INTERVAL" default:"10s"`
	ResubscribeInterval   time.Duration       `envconfig:"RESUBSCRIBE_INTERVAL" default:"10s"`
	JwksURL               string              `envconfig:"JWKS_URL"`
	OAuth                 manager.Config      `envconfig:"OAUTH"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
