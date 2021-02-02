package refImpl

import (
	"fmt"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
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
		service:      service.New(logger, config.Service, config.Database, config.Clients),
	}, nil
}

// Serve starts handling requests.
func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

// Shutdown shutdowns the service.
func (r *RefImpl) Shutdown() error {
	r.service.Shutdown()
	return nil
}
