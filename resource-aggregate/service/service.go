package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	cqrsEventBus "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	cqrsEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/kit/log"
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
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.MongoDB, logger, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	publisher, err := nats.NewPublisherV2(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		eventstore.Close(ctx)
		return nil, fmt.Errorf("cannot create kafka publisher %w", err)
	}

	service, err := NewService(config, logger, eventstore, publisher)
	if err != nil {
		eventstore.Close(ctx)
		publisher.Close()
		return nil, fmt.Errorf("cannot create kafka publisher %w", err)
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
func NewService(config Config, logger *zap.Logger, eventStore EventStore, publisher cqrsEventBus.Publisher) (*Service, error) {
	validator, err := validator.New(config.Clients.OAuthProvider.Jwks, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	auth := NewAuth(validator, config.Clients.OAuthProvider.OwnerClaim)
	streamInterceptors := []grpc.StreamServerInterceptor{}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())

	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger))
	}
	unaryInterceptors = append(unaryInterceptors, auth.Unary())
	grpcServer, err := server.New(config.APIs.GRPC.Server, logger, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
		streamInterceptors...,
	)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)))
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	grpcServer.AddCloseFunc(validator.Close)

	oauthMgr, err := manager.New(config.Clients.OAuthProvider.OAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	grpcServer.AddCloseFunc(oauthMgr.Close)

	asConn, err := client.New(config.Clients.AuthServer, logger, grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
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

	userDevicesManager := clientAS.NewUserDevicesManager(userDevicesChanged, authClient, config.APIs.GRPC.Capabilities.UserDevicesManagerTickFrequency, config.APIs.GRPC.Capabilities.UserDevicesManagerExpiration, func(err error) { log.Errorf("resource-aggregate: error occurs during receiving devices: %v", err) })
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
