package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"

	"google.golang.org/grpc"

	connectorStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/oauth/manager"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
	cancel  context.CancelFunc
	doneWg  *sync.WaitGroup
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

//New create new Server with provided store and bus
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager, store connectorStore.Store) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	listenTLSConfig := listenCertManager.GetServerTLSConfig()

	ln, err := tls.Listen("tcp", config.Addr, listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.OAuth, dialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %w", err)
	}

	raConn, err := grpc.Dial(config.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	authConn, err := grpc.Dial(config.AuthServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(authConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	_, err = url.Parse(config.OAuthCallback)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	devicesSubscription := NewDevicesSubscription(rdClient, raClient)
	err = devicesSubscription.Load(store)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	subscriptionManager := NewSubscriptionManager(config.EventsURL, asClient, raClient, store, devicesSubscription, config.OAuthCallback)
	requestHandler := NewRequestHandler(config.OAuthCallback, subscriptionManager, asClient, raClient, store)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	if !config.PullDevicesDisabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parent := ctx
			for func() bool {
				ctx, cancel := context.WithTimeout(parent, config.PullDevicesInterval)
				defer cancel()
				err := pullDevices(ctx, store, asClient, raClient, devicesSubscription, subscriptionManager, config.OAuthCallback)
				if err != nil {
					log.Errorf("cannot pull devices: %v", err)
				}
				select {
				case <-ctx.Done():
					if ctx.Err() == context.Canceled {
						return false
					}
				}
				return true
			}() {
			}
		}()
	}
	server := Server{
		server:  NewHTTP(requestHandler),
		cfg:     config,
		handler: requestHandler,
		ln:      ln,
		cancel:  cancel,
		doneWg:  &wg,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	s.doneWg.Wait()
	return s.server.Shutdown(context.Background())
}
