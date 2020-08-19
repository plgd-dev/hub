package service

import (
	"encoding/json"
	"fmt"
	"time"

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
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
