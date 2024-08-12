package service

import (
	"context"
	"fmt"

	certAuthorityPb "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcGatewayPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	m2mOAuthServerPb "github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/service"
	snippetServicePb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/reflection"
)

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	interceptor := pkgGrpc.MakeAuthInterceptors(func(ctx context.Context, _ string) (context.Context, error) {
		return ctx, nil
	})
	opts, err := server.MakeDefaultOptions(interceptor, logger, noop.NewTracerProvider())
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.APIs.GRPC.BaseConfig, fileWatcher, logger, opts...)
	if err != nil {
		return nil, err
	}

	for _, service := range config.APIs.GRPC.ReflectedServices {
		switch service {
		case grpcGatewayPb.GrpcGateway_ServiceDesc.ServiceName:
			grpcGatewayPb.RegisterGrpcGatewayServer(server.Server, &grpcGatewayPb.UnimplementedGrpcGatewayServer{})
		case certAuthorityPb.CertificateAuthority_ServiceDesc.ServiceName:
			certAuthorityPb.RegisterCertificateAuthorityServer(server.Server, &certAuthorityPb.UnimplementedCertificateAuthorityServer{})
		case snippetServicePb.SnippetService_ServiceDesc.ServiceName:
			snippetServicePb.RegisterSnippetServiceServer(server.Server, &snippetServicePb.UnimplementedSnippetServiceServer{})
		case m2mOAuthServerPb.M2MOAuthService_ServiceDesc.ServiceName:
			m2mOAuthServerPb.RegisterM2MOAuthServiceServer(server.Server, &m2mOAuthServerPb.UnimplementedM2MOAuthServiceServer{})
		}
	}
	// Register the reflection service
	reflection.Register(server.Server)

	return service.New(server), nil
}
