package service

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/net/grpc"
)

// Config represents application configuration
type Config struct {
	grpc.Config

	JwksURI string `envconfig:"JWKS_URI"  default:"https://127.0.0.1:7000/oauth/jwks"`
}

//String returns string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
