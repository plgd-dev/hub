package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/security/oauth/manager"
)

type ClientCfg struct {
	pb.ClientConfigurationResponse
	CloudCAPool string `envconfig:"CLOUD_CA_POOL" env:"CLOUD_CA_POOL" long:"cloud-ca" description:"file path to the root certificate in PEM format"`
}

// Config represent application configuration
type Config struct {
	OAuth                     manager.Config `envconfig:"OAUTH"`
	AuthServerAddr            string         `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	ResourceAggregateAddr     string         `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                      string         `envconfig:"FQDN" default:"grpcgw.ocf.cloud"`
	TimeoutForRequests        time.Duration  `envconfig:"TIMEOUT_FOR_REQUESTS"  default:"10s"`
	ClientConfiguration       ClientCfg      `envconfig:"CLIENT_CONFIGURATION"`
	ProjectionCacheExpiration time.Duration  `envconfig:"PROJECTION_CACHE_EXPIRATION" default:"1m"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
