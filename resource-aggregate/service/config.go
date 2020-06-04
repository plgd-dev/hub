package service

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/net/grpc"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr               string `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	SnapshotThreshold            int    `envconfig:"SNAPSHOT_THRESHOLD" default:"128"`
	ConcurrencyExceptionMaxRetry int    `envconfig:"OCC_MAX_RETRY" default:"8"`
	JwksURL                      string `envconfig:"JWKS_URL"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
