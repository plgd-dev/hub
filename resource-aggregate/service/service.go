package service

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pbAS "github.com/go-ocf/ocf-cloud/authorization/pb"
	cqrsEventBus "github.com/go-ocf/cqrs/eventbus"
	cqrsEventStore "github.com/go-ocf/cqrs/eventstore"
	cqrsMaintenance "github.com/go-ocf/cqrs/eventstore/maintenance"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type EventStore interface {
	cqrsEventStore.EventStore
	cqrsMaintenance.EventStore
	GetInstanceId(ctx context.Context, resourceId string) (int64, error)
	RemoveInstanceId(ctx context.Context, instanceId int64) error
}

//Server handle HTTP request
type Server struct {
	server  *kitNetGrpc.Server
	cfg     Config
	handler *RequestHandler
	sigs    chan os.Signal
}

type ClientCertManager = interface {
	GetClientTLSConfig() tls.Config
}

type ServerCertManager = interface {
	GetServerTLSConfig() tls.Config
}

// New creates new Server with provided store and publisher.
func New(config Config, clientCertManager ClientCertManager, serverCertManager ServerCertManager, eventStore EventStore, publisher cqrsEventBus.Publisher) *Server {
	clientTLSConfig := clientCertManager.GetClientTLSConfig()
	serverTLSConfig := serverCertManager.GetServerTLSConfig()

	authConn, err := grpc.Dial(config.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	grpcServer, err := kitNetGrpc.NewServer(config.Config.Addr, grpc.Creds(credentials.NewTLS(&serverTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	requestHandler := NewRequestHandler(config, eventStore, publisher, authClient)
	pb.RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

	server := Server{
		server:  grpcServer,
		cfg:     config,
		handler: requestHandler,
		sigs:    make(chan os.Signal, 1),
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
	return err
}

// Serve serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	return s.serveWithHandlingSignal(s.server.Serve)
}

// Shutdown ends serving
func (s *Server) Shutdown() {
	s.sigs <- syscall.SIGTERM
}
