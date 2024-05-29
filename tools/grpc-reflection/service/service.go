package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/reflection"
)

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {

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
		case "GrpcGateway":
			pb.RegisterGrpcGatewayServer(server.Server, &pb.UnimplementedGrpcGatewayServer{})
			// Add cases for other services here
		}
	}
	// Register the reflection service
	reflection.Register(server.Server)

	return service.New(server), nil
}
