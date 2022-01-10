package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	clientIS "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	cqrsEventBus "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	cqrsEventStore "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/maintenance"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
)

type EventStore interface {
	cqrsEventStore.EventStore
	cqrsMaintenance.EventStore
}

//Service handle GRPC request
type Service struct {
	server  *server.Server
	handler *RequestHandler
	sigs    chan os.Signal
}

func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	closeEventStore := func() {
		err := eventstore.Close(ctx)
		if err != nil {
			logger.Errorf("error occurs during closing of connection to mongodb: %w", err)
		}
	}
	naClient, err := natsClient.New(config.Clients.Eventbus.NATS.Config, logger)
	if err != nil {
		closeEventStore()
		return nil, fmt.Errorf("cannot create nats client %w", err)
	}
	publisher, err := publisher.New(naClient.GetConn(), config.Clients.Eventbus.NATS.JetStream, publisher.WithMarshaler(utils.Marshal), publisher.WithFlusherTimeout(config.Clients.Eventbus.NATS.Config.FlusherTimeout))
	if err != nil {
		naClient.Close()
		closeEventStore()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	naClient.AddCloseFunc(publisher.Close)

	service, err := NewService(ctx, config, logger, eventstore, publisher)
	if err != nil {
		naClient.Close()
		closeEventStore()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	service.AddCloseFunc(closeEventStore)
	service.AddCloseFunc(naClient.Close)

	return service, nil
}

func newGrpcServer(ctx context.Context, config GRPCConfig, logger log.Logger) (*server.Server, error) {
	validator, err := validator.New(ctx, config.Authorization.Config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	authInterceptor := server.NewAuth(validator)
	opts, err := server.MakeDefaultOptions(authInterceptor, logger)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	grpcServer, err := server.New(config.Config, logger, opts...)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	grpcServer.AddCloseFunc(validator.Close)
	return grpcServer, nil
}

func newIdentityStoreClient(config IdentityStoreConfig, logger log.Logger) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := client.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to identity-store: %w", err)
	}
	closeIsConn := func() {
		if err := isConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to identity-store: %w", err)
		}
	}
	isClient := pbIS.NewIdentityStoreClient(isConn.GRPC())
	return isClient, closeIsConn, nil
}

// New creates new Server with provided store and publisher.
func NewService(ctx context.Context, config Config, logger log.Logger, eventStore EventStore, publisher cqrsEventBus.Publisher) (*Service, error) {
	grpcServer, err := newGrpcServer(ctx, config.APIs.GRPC, logger)
	if err != nil {
		return nil, err
	}

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, logger)
	if err != nil {
		grpcServer.Close()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	grpcServer.AddCloseFunc(closeIsClient)

	nats, err := natsClient.New(config.Clients.Eventbus.NATS.Config, logger)
	if err != nil {
		grpcServer.Close()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	grpcServer.AddCloseFunc(nats.Close)

	ownerCache := clientIS.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})
	grpcServer.AddCloseFunc(ownerCache.Close)

	requestHandler := NewRequestHandler(config, eventStore, publisher, func(ctx context.Context, owner string, deviceIDs []string) ([]string, error) {
		getAllDevices := len(deviceIDs) == 0
		if !getAllDevices {
			return ownerCache.GetSelectedDevices(ctx, deviceIDs)
		}
		return ownerCache.GetDevices(ctx)
	})
	RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

	return &Service{
		server:  grpcServer,
		handler: requestHandler,
		sigs:    make(chan os.Signal, 1),
	}, nil
}

func (s *Service) serveWithHandlingSignal(serve func() error) error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(s *Service) {
		defer wg.Done()
		err = serve()
	}(s)

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs

	s.server.Close()
	wg.Wait()

	return err
}

// Serve serve starts the service's HTTP server and blocks.
func (s *Service) Serve() error {
	return s.serveWithHandlingSignal(s.server.Serve)
}

// Shutdown ends serving
func (s *Service) Shutdown() {
	s.sigs <- syscall.SIGTERM
}

func (s *Service) AddCloseFunc(f func()) {
	s.server.AddCloseFunc(f)
}
