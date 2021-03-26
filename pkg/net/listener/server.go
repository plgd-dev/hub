package listener

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"go.uber.org/zap"
)

// Server handles gRPC requests to the service.
type Server struct {
	listener  net.Listener
	closeFunc []func()
}

// NewServer instantiates a listen server.
// When passing addr with an unspecified port or ":", use Addr().
func New(config Config, logger *zap.Logger) (*Server, error) {
	certManager, err := server.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}

	lis, err := tls.Listen("tcp", config.Addr, certManager.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("listening failed: %w", err)
	}

	return &Server{listener: lis, closeFunc: []func(){certManager.Close}}, nil
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Server.
func (s *Server) AddCloseFunc(f func()) {
	s.closeFunc = append(s.closeFunc, f)
}

func (s *Server) Close() error {
	err := s.listener.Close()
	for _, f := range s.closeFunc {
		f()
	}
	return err
}

func (s *Server) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}
