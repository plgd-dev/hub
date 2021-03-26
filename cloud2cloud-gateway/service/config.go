package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
)

//Config represent application configuration
type Config struct {
	Addr                  string         `envconfig:"ADDRESS" env:"ADDRESS" long:"address" default:"0.0.0.0:9100"`
	ResourceDirectoryAddr string         `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                  string         `envconfig:"FQDN" default:"cloud2cloud.pluggedin.cloud"`
	ReconnectInterval     time.Duration  `envconfig:"RECONNECT_INTERVAL" default:"10s"`
	JwksURL               string         `envconfig:"JWKS_URL"`
	OAuth                 manager.Config `envconfig:"OAUTH"`
	EmitEventTimeout      time.Duration  `envconfig:"EMIT_EVENT_TIMEOUT" default:"5s"`
	ResourceAggregateAddr string         `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	GoRoutinePoolSize     int            `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	OwnerClaim            string         `envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
	Nats                  nats.Config
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
