package server

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func New(config Config, logger *zap.Logger, opts ...grpc.ServerOption) (*Server, error) {
	tls, err := server.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}

	v := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tls.GetTLSConfig())),
	}
	if len(v) > 0 {
		v = append(v, opts...)
	}

	server, err := NewServer(config.Addr, v...)
	if err != nil {
		return nil, err
	}
	server.AddCloseFunc(func() {
		tls.Close()
	})
	return server, nil
}
