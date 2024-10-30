package grpc

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	*server.Server
}

func New(config Config, clientApplicationServer *CertificateAuthorityServer, validator *validator.Validator, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Service, error) {
	opts, err := server.MakeDefaultOptions(server.NewAuth(validator), logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.BaseConfig, fileWatcher, logger, tracerProvider, opts...)
	if err != nil {
		return nil, err
	}
	pb.RegisterCertificateAuthorityServer(server.Server, clientApplicationServer)

	// CertificateAuthority needs to stop gracefully to ensure that all commands are processed.
	server.SetGracefulStop(true)

	return &Service{
		Server: server,
	}, nil
}
