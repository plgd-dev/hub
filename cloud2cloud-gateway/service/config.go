package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/oauth/manager"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	ResourceDirectoryAddr string         `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                  string         `envconfig:"FQDN" default:"cloud2cloud.pluggedin.cloud"`
	ReconnectInterval     time.Duration  `envconfig:"RECONNECT_INTERVAL" default:"10s"`
	JwksURL               string         `envconfig:"JWKS_URL"`
	OAuth                 manager.Config `envconfig:"OAUTH"`
	EmitEventTimeout      time.Duration  `envconfig:"EMIT_EVENT_TIMEOUT" default:"5s"`
	ResourceAggregateAddr string         `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	GoRoutinePoolSize     int            `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	Nats                  nats.Config
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
