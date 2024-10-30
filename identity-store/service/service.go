package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/config"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/cqldb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type Persistence = interface {
	NewTransaction(ctx context.Context) persistence.PersistenceTx
	Clear(ctx context.Context) error
	Close(ctx context.Context) error
}

func newPersistence(ctx context.Context, config config.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (Persistence, error) {
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("cqldb: %w", err)
		}
		return s, nil
	}
	return nil, fmt.Errorf("invalid eventstore use('%v')", config.Use)
}

// Service holds dependencies of IdentityStore.
type Service struct {
	pb.UnimplementedIdentityStoreServer
	persistence Persistence
	publisher   *publisher.Publisher
	ownerClaim  string
	hubID       string
	logger      log.Logger
}

// Server is an HTTP server for the Service.
type Server struct {
	service    *Service
	grpcServer *server.Server
	cfg        Config
}

func NewService(persistence Persistence, publisher *publisher.Publisher, ownerClaim, hubID string, logger log.Logger) *Service {
	return &Service{
		persistence: persistence,
		ownerClaim:  ownerClaim,
		publisher:   publisher,
		logger:      logger,
		hubID:       hubID,
	}
}

func NewServer(ctx context.Context, cfg Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, publisher *publisher.Publisher, grpcOpts ...grpc.ServerOption) (*Server, error) {
	grpcServer, err := server.New(cfg.APIs.GRPC.BaseConfig, fileWatcher, logger, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc listener: %w", err)
	}

	persistence, err := newPersistence(ctx, cfg.Clients.Storage, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create connector to persistent storage: %w", err)
	}
	grpcServer.AddCloseFunc(func() {
		if err := persistence.Close(ctx); err != nil {
			log.Debugf("failed to close persistent storage: %w", err)
		}
	})

	service := NewService(persistence, publisher, cfg.APIs.GRPC.Authorization.OwnerClaim, cfg.HubID, logger)

	pb.RegisterIdentityStoreServer(grpcServer.Server, service)

	return &Server{service: service, grpcServer: grpcServer, cfg: cfg}, nil
}

// New creates the service's HTTP server.
func New(ctx context.Context, cfg Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, cfg.Clients.OpenTelemetryCollector, "identity-store", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	tracerProvider := otelClient.GetTracerProvider()

	naClient, err := client.New(cfg.Clients.Eventbus.NATS.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create nats client %w", err)
	}
	publisher, err := publisher.New(naClient.GetConn(), cfg.Clients.Eventbus.NATS.JetStream, publisher.WithMarshaler(utils.Marshal))
	if err != nil {
		otelClient.Close()
		naClient.Close()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	naClient.AddCloseFunc(otelClient.Close)
	naClient.AddCloseFunc(publisher.Close)
	validator, err := validator.New(ctx, cfg.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		naClient.Close()
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	interceptor := server.NewAuth(validator, server.WithDisabledTokenForwarding())
	opts, err := server.MakeDefaultOptions(interceptor, logger, tracerProvider)
	if err != nil {
		validator.Close()
		naClient.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	s, err := NewServer(ctx, cfg, fileWatcher, logger, tracerProvider, publisher, opts...)
	if err != nil {
		validator.Close()
		naClient.Close()
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	s.grpcServer.AddCloseFunc(validator.Close)
	s.grpcServer.AddCloseFunc(naClient.Close)

	// IdentityStore needs to stop gracefully to ensure that all commands are processed.
	s.grpcServer.SetGracefulStop(true)

	return service.New(s), nil
}

// Serve starts the service's GRPC and HTTP server and blocks.
func (s *Server) Serve() error {
	return s.grpcServer.Serve()
}

// Shutdown ends serving
func (s *Server) Close() error {
	return s.grpcServer.Close()
}
