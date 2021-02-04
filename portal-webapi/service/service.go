package service

import (
	"crypto/tls"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"net"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
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
	GetTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetTLSConfig() *tls.Config
}

//New create new Server with provided stores
func New(config Config, dialCertManager *client.CertManager, listenCertManager *server.CertManager) *Server {
	dialTLSConfig := dialCertManager.GetTLSConfig()
	listenTLSConfig := listenCertManager.GetTLSConfig()
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

	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	server := Server{
		config: config,
		ln:     ln,
	}

	requestHandler := NewRequestHandler(&server, raClient, rdClient)
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
