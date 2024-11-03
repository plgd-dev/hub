package server

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func New(config BaseConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, customVerify pkgX509.CustomDistributionPointVerification, opts ...grpc.ServerOption) (*Server, error) {
	err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	tls, err := server.New(config.TLS, fileWatcher, logger, tracerProvider, general.WithCustomDistributionPointVerification(customVerify))
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}

	v := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tls.GetTLSConfig())),
		grpc.KeepaliveEnforcementPolicy(config.EnforcementPolicy.ToGrpc()),
		grpc.KeepaliveParams(config.KeepAlive.ToGrpc()),
		grpc.MaxRecvMsgSize(config.RecvMsgSize),
		grpc.MaxSendMsgSize(config.SendMsgSize),
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
