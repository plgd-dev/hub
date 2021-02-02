package refImpl

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

type RefImpl struct {
	service           *kitNetGrpc.Server
}

func Init(config service.Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	server, err := service.NewService(logger, config)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	return &RefImpl{
		service:          server,
	}, nil
}

// Serve starts handling requests.
func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

// Shutdown shutdowns the service.
func (r *RefImpl) Shutdown() error {
	r.service.Close()
	return nil
}
