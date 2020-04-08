package service

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"

	"github.com/go-ocf/kit/net/grpc"
)

//Config represent application configuration
type Config struct {
	grpc.Config
	AuthServerAddr        string `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
	ResourceAggregateAddr string `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	FQDN                  string `envconfig:"FQDN" default:"openapi.pluggedin.cloud"`
	OAuthCallback         string `envconfig:"OAUTH_CALLBACK" required:"true"`
	EventsURL             string `envconfig:"EVENTS_URL" required:"true"`
	OriginCloud           store.LinkedCloud
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
