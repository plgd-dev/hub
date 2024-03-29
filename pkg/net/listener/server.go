package listener

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
)

// Server handles gRPC requests to the service.
type Server struct {
	listener  net.Listener
	closeFunc fn.FuncList
}

// NewServer instantiates a listen server.
// When passing addr with an unspecified port or ":", use Addr().
func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Server, error) {
	certManager, err := server.New(config.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}

	lis, err := tls.Listen("tcp", config.Addr, certManager.GetTLSConfig())
	if err != nil {
		certManager.Close()
		return nil, fmt.Errorf("listening failed: %w", err)
	}
	s := &Server{listener: lis}
	s.AddCloseFunc(certManager.Close)
	return s, nil
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Server.
func (s *Server) AddCloseFunc(f func()) {
	s.closeFunc.AddFunc(f)
}

func (s *Server) Close() error {
	err := s.listener.Close()
	s.closeFunc.Execute()
	return err
}

func (s *Server) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}
