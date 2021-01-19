package service

import (
	"context"
	"crypto/tls"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"

	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
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
	cqrsEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/jwt"
	"github.com/plgd-dev/kit/security/oauth/manager"
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

func NewNumParallelProcessedRequestLimiter(n int) *NumParallelProcessedRequestLimiter {
	return &NumParallelProcessedRequestLimiter{
		w: semaphore.NewWeighted(int64(n)),
	}
}

func (l *NumParallelProcessedRequestLimiter) StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := l.w.Acquire(stream.Context(), 1)
	if err != nil {
		return err
	}
	defer l.w.Release(1)
	wrapped := grpc_middleware.WrapServerStream(stream)
	return handler(srv, wrapped)
}

func (l *NumParallelProcessedRequestLimiter) UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := l.w.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}
	defer l.w.Release(1)
	return handler(ctx, req)
}

// New creates new Server with provided store and publisher.
func New(logger *zap.Logger, service APIsConfig, database Database, clients ClientsConfig ) *Server {

	mongoCertManager, err := client.New(database.MongoDB.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create mongodb client cert manager %w", err)
	}
	eventStore, err := mongodb.NewEventStore(database.MongoDB, nil, mongodb.WithTLS(mongoCertManager.GetTLSConfig()))
	if err != nil {
		log.Errorf("cannot create mongodb eventstore %w", err)
	}

	natsCertManager, err := client.New(clients.Nats.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create nats client cert manager %w", err)
	}
	publisher, err := nats.NewPublisher(clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	if err != nil {
		log.Errorf("cannot create kafka publisher %w", err)
	}

	oauthCertManager, err := client.New(clients.OAuth.OAuthTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create oauth client cert manager %w", err)
	}
	auth := NewAuth(clients.OAuth.JwksURL, oauthCertManager.GetTLSConfig())
	rateLimiter := NewNumParallelProcessedRequestLimiter(service.RA.Capabilities.NumParallelRequest)
	streamInterceptors := []grpc.StreamServerInterceptor{
		rateLimiter.StreamServerInterceptor,
	}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		rateLimiter.UnaryServerInterceptor,
	}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger))
	}
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	grpcCertManager, err := server.New(service.RA.GrpcTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create server cert manager %w", err)
	}
	grpcServer, err := kitNetGrpc.NewServer(service.RA.GrpcAddr, grpc.Creds(credentials.NewTLS(grpcCertManager.GetTLSConfig())),
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

	oauthMgr, err := manager.NewManagerFromConfiguration(clients.OAuth.OAuth, oauthCertManager.GetTLSConfig())
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	asCertManager, err := client.New(clients.AuthServer.AuthTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create oauth client cert manager %w", err)
	}
	asConn, err := grpc.Dial(clients.AuthServer.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		log.Fatalf("cannot connect to authorization server: %v", err)
	}
	authClient := pbAS.NewAuthorizationServiceClient(asConn)

	userDevicesManager := clientAS.NewUserDevicesManager(userDevicesChanged, authClient,
		service.RA.Capabilities.UserDevicesManagerTickFrequency, service.RA.Capabilities.UserDevicesManagerExpiration,
		func(err error) { log.Errorf("resource-aggregate: error occurs during receiving devices: %v", err) })
		requestHandler := NewRequestHandler(service, eventStore, publisher, func(ctx context.Context, userID, deviceID string) (bool, error) {
		devices, err := userDevicesManager.GetUserDevices(ctx, userID)
		if err != nil {
			return false, err
		}
		for _, id := range devices {
			if id == deviceID {
				return true, nil
			}
		}
		devices, err = userDevicesManager.UpdateUserDevices(ctx, userID)
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
	pb.RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

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
	s.oauthCertManager.Close()
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

func NewAuth(jwksUrl string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor: %v", err)
			return ctx, err
		}
		userID, err := kitNetGrpc.UserIDFromMD(ctx)
		if err != nil {
			userID, err = kitNetGrpc.UserIDFromTokenMD(ctx)
			if err == nil {
				ctx = kitNetGrpc.CtxWithIncomingUserID(ctx, userID)
			}
		}
		if err != nil {
			log.Errorf("auth cannot get userID: %v", err)
			return ctx, err
		}
		return kitNetGrpc.CtxWithUserID(ctx, userID), nil
	})
}
