package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/net/grpc"
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
	return config.ToString(c)
}
