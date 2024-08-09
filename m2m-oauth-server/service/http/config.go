package http

import (
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

type Config struct {
	Connection    listener.Config  `yaml:",inline" json:",inline"`
	Authorization validator.Config `yaml:"authorization" json:"authorization"`
	Server        server.Config    `yaml:",inline" json:",inline"`
}
