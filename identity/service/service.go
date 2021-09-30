package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/identity/pb"
	"github.com/plgd-dev/cloud/identity/persistence"
	"github.com/plgd-dev/cloud/identity/persistence/mongodb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc"
)

type Persistence = interface {
	NewTransaction(ctx context.Context) persistence.PersistenceTx
	Clear(ctx context.Context) error
	Close(ctx context.Context) error
}

// Service holds dependencies of the Identity service.
type Service struct {
	pb.UnimplementedIdentityServiceServer
	persistence Persistence
	publisher   *publisher.Publisher
	ownerClaim  string
}

// Server is an HTTP server for the Service.
type Server struct {
	service    *Service
	grpcServer *server.Server
	cfg        Config
}

func NewService(persistence Persistence, publisher *publisher.Publisher, ownerClaim string) *Service {
	return &Service{
		persistence: persistence,
		ownerClaim:  ownerClaim,
		publisher:   publisher,
	}
}

func NewServer(ctx context.Context, cfg Config, logger log.Logger, publisher *publisher.Publisher, grpcOpts ...grpc.ServerOption) (*Server, error) {
	grpcServer, err := server.New(cfg.APIs.GRPC, logger, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc listener: %w", err)
	}

	persistence, err := mongodb.NewStore(ctx, cfg.Clients.Storage.MongoDB, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create connector to mongo: %w", err)
	}
	grpcServer.AddCloseFunc(func() {
		if err := persistence.Close(ctx); err != nil {
			log.Debugf("failed to close mongodb connector: %w", err)
		}
	})

	service := NewService(persistence, publisher, cfg.APIs.GRPC.Authorization.OwnerClaim)

	pb.RegisterIdentityServiceServer(grpcServer.Server, service)

	return &Server{service: service, grpcServer: grpcServer, cfg: cfg}, nil
}

// New creates the service's HTTP server.
func New(ctx context.Context, cfg Config, logger log.Logger) (*Server, error) {
	naClient, err := client.New(cfg.Clients.Eventbus.NATS.Config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client %w", err)
	}
	publisher, err := publisher.New(naClient.GetConn(), cfg.Clients.Eventbus.NATS.JetStream, publisher.WithMarshaler(utils.Marshal))
	if err != nil {
		naClient.Close()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	naClient.AddCloseFunc(publisher.Close)
	validator, err := validator.New(ctx, cfg.APIs.GRPC.Authorization.Config, logger)
	if err != nil {
		naClient.Close()
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	interceptor := server.NewAuth(validator, server.WithDisabledTokenForwarding())
	opts, err := server.MakeDefaultOptions(interceptor, logger)
	if err != nil {
		validator.Close()
		naClient.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	s, err := NewServer(ctx, cfg, logger, publisher, opts...)
	if err != nil {
		validator.Close()
		naClient.Close()
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	s.grpcServer.AddCloseFunc(validator.Close)
	s.grpcServer.AddCloseFunc(naClient.Close)
	return s, nil
}

// Serve starts the service's GRPC and HTTP server and blocks.
func (s *Server) Serve() error {
	return s.grpcServer.Serve()
}

// Shutdown ends serving
func (s *Server) Shutdown() {
	s.grpcServer.Close()
}
