package service

import (
	"context"
	"crypto/tls"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
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
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	cqrsEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"

	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type EventStore interface {
	cqrsEventStore.EventStore
	cqrsMaintenance.EventStore
}

//Server handle HTTP request
type Server struct {
	server             *kitNetGrpc.Server
	handler            *RequestHandler
	sigs               chan os.Signal
	authConn           *grpc.ClientConn
	userDevicesManager *clientAS.UserDevicesManager

	eventstore         *mongodb.EventStore
	publisher          *nats.Publisher

	grpcCertManager    *server.CertManager
	mongoCertManager   *client.CertManager
	natsCertManager    *client.CertManager
	oauthCertManager   *client.CertManager
	asCertManager      *client.CertManager
}

type ClientCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ServerCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

type NumParallelProcessedRequestLimiter struct {
	w *semaphore.Weighted
}

// New creates new Server with provided store and publisher.
func New(config Config, logger *zap.Logger) *Server {
	mongoCertManager, err := client.New(config.Database.MongoDB.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create mongodb client cert manager %w", err)
	}
	eventStore, err := mongodb.NewEventStore(context.Background(), config.Database.MongoDB, nil, mongodb.WithTLS(mongoCertManager.GetTLSConfig()))
	if err != nil {
		log.Errorf("cannot create mongodb eventstore %w", err)
	}

	natsCertManager, err := client.New(config.Clients.Nats.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create nats client cert manager %w", err)
	}
	publisher, err := nats.NewPublisher(config.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	if err != nil {
		log.Errorf("cannot create kafka publisher %w", err)
	}

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err = config.Clients.OAuthProvider.TLSConfig.Validate()
	if err != nil {
		log.Errorf("failed to validate client tls config: %v", err)
	} else {
		oauthCertManager, err := client.New(config.Clients.OAuthProvider.TLSConfig, logger)
		if err != nil {
			log.Errorf("cannot create oauth client cert manager %v", err)
		} else {
			oauthTLSConfig = oauthCertManager.GetTLSConfig()
		}
	}

	auth := NewAuth(config.Clients.OAuthProvider.JwksURL, config.Clients.OAuthProvider.OwnerClaim, oauthTLSConfig)
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

	grpcCertManager, err := server.New(config.Service.Grpc.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create grpc server cert manager %v", err)
	}
	grpcServer, err := kitNetGrpc.NewServer(config.Service.Grpc.Addr, grpc.Creds(credentials.NewTLS(grpcCertManager.GetTLSConfig())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.Clients.OAuthProvider.OAuth, oauthTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	asCertManager, err := client.New(config.Clients.Authorization.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create as client cert manager %w", err)
	}
	asConn, err := grpc.Dial(config.Clients.Authorization.Addr, grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		log.Fatalf("cannot connect to authorization server: %v", err)
	}
	authClient := pbAS.NewAuthorizationServiceClient(asConn)

	userDevicesManager := clientAS.NewUserDevicesManager(userDevicesChanged, authClient, config.Service.Grpc.Capabilities.UserDevicesManagerTickFrequency, config.Service.Grpc.Capabilities.UserDevicesManagerExpiration, func(err error) { log.Errorf("resource-aggregate: error occurs during receiving devices: %v", err) })
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
	RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

	server := Server{
		server:             grpcServer,
		handler:            requestHandler,
		sigs:               make(chan os.Signal, 1),
		authConn:           asConn,
		userDevicesManager: userDevicesManager,

		eventstore:         eventStore,
		publisher:          publisher,

		grpcCertManager:    grpcCertManager,
		mongoCertManager:   mongoCertManager,
		natsCertManager:    natsCertManager,
		asCertManager:      asCertManager,
		oauthCertManager:   oauthCertManager,
	}

	return &server
}

func (s *Server) serveWithHandlingSignal(serve func() error) error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(s *Server) {
		defer wg.Done()
		err = serve()
	}(s)

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs

	s.server.Stop()
	wg.Wait()
	s.userDevicesManager.Close()
	s.authConn.Close()

	s.eventstore.Close(context.Background())
	s.publisher.Close()
	if s.oauthCertManager != nil { s.oauthCertManager.Close() }
	s.asCertManager.Close()
	s.grpcCertManager.Close()
	s.natsCertManager.Close()
	s.mongoCertManager.Close()

	return err
}

// Serve serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	return s.serveWithHandlingSignal(s.server.Serve)
}

// Shutdown ends serving
func (s *Server) Shutdown() {
	s.sigs <- syscall.SIGTERM
	return
}

func NewAuth(jwksUrl, ownerClaim string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
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
