package service

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/cloud/authorization/oauth"
	"github.com/go-ocf/cloud/authorization/persistence/mongodb"
	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/certManager"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log log.Config

	Device provider.Config `envconfig:"DEVICE" env:"DEVICE"`
	SDK    oauth.Config    `envconfig:"SDK_OAUTH" env:"SDK_OAUTH"`

	MongoDB  mongodb.Config     `envconfig:"MONGODB" env:"MONGODB"`
	Listen   certManager.Config `envconfig:"LISTEN" env:"LISTEN"`
	Dial     certManager.Config `envconfig:"DIAL" env:"DIAL"`
	Addr     string             `envconfig:"ADDRESS" env:"ADDRESS" default:"0.0.0.0:9100"`
	HTTPAddr string             `envconfig:"HTTP_ADDRESS" env:"HTTP_ADDRESS" default:"0.0.0.0:9200"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
