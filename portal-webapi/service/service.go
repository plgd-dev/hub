package service

import (
	"crypto/tls"
	"net"

	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/kit/log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//Server handle HTTP request
type Server struct {
	server *fasthttp.Server
	config Config
	ln     net.Listener
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

//New create new Server with provided stores
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	listenTLSConfig.ClientAuth = tls.NoClientCert

	ln, err := tls.Listen("tcp", config.Addr, listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	raConn, err := grpc.Dial(config.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)
	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	rdClient := pbRD.NewResourceDirectoryClient(rdConn)
	rsClient := pbRS.NewResourceShadowClient(rdConn)
	ddClient := pbDD.NewDeviceDirectoryClient(rdConn)

	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	server := Server{
		config: config,
		ln:     ln,
	}

	requestHandler := NewRequestHandler(&server, raClient, rsClient, rdClient, ddClient)
	router := NewHTTP(requestHandler)

	server.server = &fasthttp.Server{
		Handler: router.Handler,
	}
	return &server
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	return s.ln.Close()
}

// Serve starts server and handle OS signals.
func (s *Server) Serve() error {
	return s.server.Serve(s.ln)
}
