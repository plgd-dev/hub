package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
)

//Config represent application configuration
type Config struct {
	Addr                            string         `envconfig:"ADDRESS" env:"ADDRESS" long:"address" default:"0.0.0.0:9100"`
	AuthServerAddr                  string         `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	SnapshotThreshold               int            `envconfig:"SNAPSHOT_THRESHOLD" default:"16"`
	ConcurrencyExceptionMaxRetry    int            `envconfig:"OCC_MAX_RETRY" default:"8"`
	JwksURL                         string         `envconfig:"JWKS_URL"`
	OAuth                           manager.Config `envconfig:"OAUTH"`
	UserDevicesManagerTickFrequency time.Duration  `envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration  `envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
	OwnerClaim                      string         `envconfig:"OWNER_CLAIM" env:"OWNER_CLAIM" default:"sub"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
