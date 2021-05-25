package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"go.uber.org/zap"

	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	cqrsEventBus "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	cqrsEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"google.golang.org/grpc"
)

type EventStore interface {
	cqrsEventStore.EventStore
	cqrsMaintenance.EventStore
}

//Service handle GRPC request
type Service struct {
	server             *server.Server
	cfg                Config
	handler            *RequestHandler
	sigs               chan os.Signal
	authConn           *grpc.ClientConn
	userDevicesManager *clientAS.UserDevicesManager
}

func New(ctx context.Context, config Config, logger *zap.Logger) (*Service, error) {
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	publisher, err := publisher.New(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		eventstore.Close(ctx)
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}

	service, err := NewService(ctx, config, logger, eventstore, publisher)
	if err != nil {
		eventstore.Close(ctx)
		publisher.Close()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	service.AddCloseFunc(func() {
		err := eventstore.Close(ctx)
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to mongodb: %w", err)
		}
	})
	service.AddCloseFunc(publisher.Close)

	return service, nil
}

// New creates new Server with provided store and publisher.
func NewService(ctx context.Context, config Config, logger *zap.Logger, eventStore EventStore, publisher cqrsEventBus.Publisher) (*Service, error) {
	validator, err := validator.New(ctx, config.APIs.GRPC.Authorization, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	opts, err := server.MakeDefaultOptions(server.NewAuth(validator, server.WithOwnerClaim(config.Clients.AuthServer.OwnerClaim)), logger)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	grpcServer, err := server.New(config.APIs.GRPC, logger, opts...)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	grpcServer.AddCloseFunc(validator.Close)

	oauthMgr, err := manager.New(config.Clients.AuthServer.OAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	grpcServer.AddCloseFunc(oauthMgr.Close)

	asConn, err := client.New(config.Clients.AuthServer.Connection, logger, grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	grpcServer.AddCloseFunc(func() {
		err := asConn.Close()
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to authorization server: %w", err)
		}
	})

	authClient := pbAS.NewAuthorizationServiceClient(asConn.GRPC())

	userDevicesManager := clientAS.NewUserDevicesManager(userDevicesChanged, authClient, config.Clients.AuthServer.PullFrequency, config.Clients.AuthServer.CacheExpiration, func(err error) { log.Errorf("resource-aggregate: error occurs during receiving devices: %v", err) })
	requestHandler := NewRequestHandler(config, eventStore, publisher, func(ctx context.Context, owner, deviceID string) (bool, error) {
		ok := userDevicesManager.IsUserDevice(owner, deviceID)
		if ok {
			return ok, nil
		}
		devices, err := userDevicesManager.UpdateUserDevices(ctx, owner)
		if err != nil {
			return false, err
		}
		for _, id := range devices {
			if id == deviceID {
				return true, nil
			}
		}
		return false, nil
	})
	grpcServer.AddCloseFunc(userDevicesManager.Close)
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

func NewAuth(validator kitNetGrpc.Validator, ownerClaim string) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWTWithValidator(validator, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor: %v", err)
			return ctx, err
		}
		owner, err := kitNetGrpc.OwnerFromMD(ctx)
		if err != nil {
			owner, err = kitNetGrpc.OwnerFromTokenMD(ctx, ownerClaim)
			if err == nil {
				ctx = kitNetGrpc.CtxWithIncomingOwner(ctx, owner)
			}
		}
		if err != nil {
			log.Errorf("auth cannot get owner: %v", err)
			return ctx, err
		}
		return kitNetGrpc.CtxWithOwner(ctx, owner), nil
	})
}
