package refImpl

import (
	"github.com/plgd-dev/kit/config"

	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/kit/log"
)

type Config struct {
	Log              log.Config            	`yaml:"log" json:"log"`
	Service          service.ServiceConfig	`yaml:"apis" json:"apis"`
	Clients			 service.ClientsConfig  `yaml:"clients" json:"clients"`
}

type RefImpl struct {
	service           *service.Server
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

// Init creates reference implementation for coap-gateway with default authorization interceptor.
func Init(config Config) (*RefImpl, error) {
	return &RefImpl{
		service:           service.New(config.Service, config.Clients),
	}, nil
}

// Serve starts handling coap requests.
func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

// Shutdown shutdowns the service.
func (r *RefImpl) Shutdown() error {
	err := r.service.Shutdown()
	return err
}
