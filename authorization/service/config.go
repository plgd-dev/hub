package service

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log log.Config				`envconfig:"LOG" long:"log"`

	Device provider.Config 		`envconfig:"DEVICE" env:"DEVICE" long:"device"`
	SDK    oauth.Config    		`envconfig:"SDK_OAUTH" env:"SDK_OAUTH" long:"sdk_oauth"`

	MongoDB  mongodb.Config     `envconfig:"MONGODB" env:"MONGODB" long:"mongodb"`
	Listen   certManager.Config `envconfig:"LISTEN" env:"LISTEN"`
	Dial     certManager.Config `envconfig:"DIAL" env:"DIAL"`
	Addr     string             `envconfig:"ADDRESS" env:"ADDRESS" long:"address" default:"0.0.0.0:9081"`
	HTTPAddr string             `envconfig:"HTTP_ADDRESS" env:"HTTP_ADDRESS" long:"http_address" default:"0.0.0.0:9085"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
