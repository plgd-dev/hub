package refImpl

import (
	"fmt"
	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/kit/log"
)

type RefImpl struct {
	service           *service.Server
}

func (c Config) CheckForDefaults() Config {
	c.Service = c.Service.CheckForDefaults()
	return c
}

// Init creates reference implementation for coap-gateway with default authorization interceptor.
func Init(config service.Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	return &RefImpl{
		service:           service.New(logger, config.Service, config.Clients),
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
