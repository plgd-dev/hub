package refImpl

import (
	"fmt"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/service"
	"github.com/plgd-dev/kit/log"
)

type RefImpl struct {
	service           *service.Server
}

func Init(config service.Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	return &RefImpl{
		service:           service.New(config, logger),
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
