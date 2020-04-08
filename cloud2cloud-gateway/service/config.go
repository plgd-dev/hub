package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-ocf/kit/net/grpc"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr        string        `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	ResourceAggregateAddr string        `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	ResourceDirectoryAddr string        `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                  string        `envconfig:"FQDN" default:"cloud2cloud.pluggedin.cloud"`
	DevicesCheckInterval  time.Duration `envconfig:"ALL_DEVICES_CHECK_INTERVAL" default:"3s"`
	TimeoutForRequests    time.Duration `envconfig:"TIMEOUT_FOR_REQUESTS"  default:"10s"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
