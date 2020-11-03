package service

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certificateManager"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log log.Config

	Device provider.Config `envconfig:"DEVICE" env:"DEVICE"`
	SDK    oauth.Config    `envconfig:"SDK_OAUTH" env:"SDK_OAUTH"`

	MongoDB  mongodb.Config            `envconfig:"MONGODB" env:"MONGODB"`
	Listen   certificateManager.Config `envconfig:"LISTEN" env:"LISTEN"`
	Dial     certificateManager.Config `envconfig:"DIAL" env:"DIAL"`
	Addr     string                    `envconfig:"ADDRESS" env:"ADDRESS" default:"0.0.0.0:9100"`
	HTTPAddr string                    `envconfig:"HTTP_ADDRESS" env:"HTTP_ADDRESS" default:"0.0.0.0:9200"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
