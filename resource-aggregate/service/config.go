package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/kit/net/grpc"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr                  string         `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	SnapshotThreshold               int            `envconfig:"SNAPSHOT_THRESHOLD" default:"16"`
	ConcurrencyExceptionMaxRetry    int            `envconfig:"OCC_MAX_RETRY" default:"8"`
	JwksURL                         string         `envconfig:"JWKS_URL"`
	OAuth                           manager.Config `envconfig:"OAUTH"`
	UserDevicesManagerTickFrequency time.Duration  `envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
	NumParallelRequest              int            `envconfig:"NUM_PARALLEL_REQUEST" default:"8"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
