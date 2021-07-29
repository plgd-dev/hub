package server

import (
	"fmt"

	"github.com/plgd-dev/cloud/pkg/log"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func New(config Config, logger log.Logger, opts ...grpc.ServerOption) (*Server, error) {
	tls, err := server.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}

	v := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tls.GetTLSConfig())),
		grpc.KeepaliveEnforcementPolicy(config.EnforcementPolicy.ToGrpc()),
		grpc.KeepaliveParams(config.KeepAlive.ToGrpc()),
	}
	if len(opts) > 0 {
		v = append(v, opts...)
	}
	server, err := NewServer(config.Addr, v...)
	if err != nil {
		tls.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	server.AddCloseFunc(tls.Close)

	return server, nil
}
