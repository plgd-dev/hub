package service

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/net/grpc"
)

// Config represent application configuration
type Config struct {
	grpc.Config

	FQDN           string `envconfig:"FQDN" default:"grpc-gateway"`
	AuthServerAddr string `envconfig:"AUTH_SERVER_ADDRESS" default:"127.0.0.1:9100"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
