package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/oauth/manager"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr        string         `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	ResourceAggregateAddr string         `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	ResourceDirectoryAddr string         `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                  string         `envconfig:"FQDN" default:"cloud2cloud.pluggedin.cloud"`
	OAuthCallback         string         `envconfig:"OAUTH_CALLBACK" required:"true"`
	EventsURL             string         `envconfig:"EVENTS_URL" required:"true"`
	PullDevicesDisabled   bool           `envconfig:"PULL_DEVICES_DISABLED" default:"false"`
	PullDevicesInterval   time.Duration  `envconfig:"PULL_DEVICES_INTERVAL" default:"5s"`
	OAuth                 manager.Config `envconfig:"OAUTH"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
