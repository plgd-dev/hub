package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	isClient "github.com/plgd-dev/cloud/identity/client"
	pbIS "github.com/plgd-dev/cloud/identity/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	naClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc"
)

type closeFunc []func()

func (s closeFunc) Close() {
	if len(s) == 0 {
		return
	}
	for _, f := range s {
		f()
	}
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	authorizationClient     pbIS.IdentityServiceClient
	resourceDirectoryClient pb.GrpcGatewayClient
	resourceAggregateClient *raClient.Client
	resourceSubscriber      *subscriber.Subscriber
	ownerCache              *isClient.OwnerCache
	config                  Config
	closeFunc               closeFunc
}

func AddHandler(ctx context.Context, svr *server.Server, config Config, logger log.Logger, goroutinePoolGo func(func()) error) error {
	handler, err := NewRequestHandlerFromConfig(ctx, config, logger, goroutinePoolGo)
	if err != nil {
		return err
	}
	svr.AddCloseFunc(handler.Close)
	pb.RegisterGrpcGatewayServer(svr.Server, handler)
	return nil
}

// Register registers the handler instance with a gRPC server.
func Register(server *grpc.Server, handler *RequestHandler) {
	pb.RegisterGrpcGatewayServer(server, handler)
}

func NewRequestHandlerFromConfig(ctx context.Context, config Config, logger log.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc closeFunc

	natsClient, err := naClient.New(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	closeFunc = append(closeFunc, natsClient.Close)

	resourceSubscriber, err := subscriber.New(natsClient.GetConn(),
		config.Clients.Eventbus.NATS.PendingLimits,
		logger,
		subscriber.WithGoPool(goroutinePoolGo),
		subscriber.WithUnmarshaler(utils.Unmarshal),
	)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}
	closeFunc = append(closeFunc, resourceSubscriber.Close)

	authorizationConn, err := client.New(config.Clients.AuthServer.Connection, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create connection to authorization server: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := authorizationConn.Close()
		if err != nil {
			logger.Errorf("error occurs during close connection to authorization server: %w", err)
		}
	})
	authorizationClient := pbIS.NewIdentityServiceClient(authorizationConn.GRPC())

	ownerCache := isClient.NewOwnerCache(config.Clients.AuthServer.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration, natsClient.GetConn(), authorizationClient, func(err error) {
		log.Errorf("error occurs during processing event by ownerCache: %v", err)
	})
	closeFunc = append(closeFunc, ownerCache.Close)

	rdConn, err := client.New(config.Clients.ResourceDirectory.Connection, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot connect to resource-directory: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := rdConn.Close()
		if err != nil {
			logger.Errorf("error occurs during close connection to resource-directory: %w", err)
		}
	})
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn.GRPC())

	raConn, err := client.New(config.Clients.ResourceAggregate.Connection, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot connect to resource-aggregate: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := raConn.Close()
		if err != nil {
			logger.Errorf("error occurs during close connection to resource-aggregate: %w", err)
		}
	})
	resourceAggregateClient := raClient.New(raConn.GRPC(), resourceSubscriber)

	return NewRequestHandler(
		authorizationClient,
		resourceDirectoryClient,
		resourceAggregateClient,
		resourceSubscriber,
		ownerCache,
		config,
		closeFunc,
	), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	authorizationClient pbIS.IdentityServiceClient,
	resourceDirectoryClient pb.GrpcGatewayClient,
	resourceAggregateClient *raClient.Client,
	resourceSubscriber *subscriber.Subscriber,
	ownerCache *isClient.OwnerCache,
	config Config,
	closeFunc closeFunc,
) *RequestHandler {
	return &RequestHandler{
		authorizationClient:     authorizationClient,
		resourceDirectoryClient: resourceDirectoryClient,
		resourceAggregateClient: resourceAggregateClient,
		resourceSubscriber:      resourceSubscriber,
		ownerCache:              ownerCache,
		config:                  config,
		closeFunc:               closeFunc,
	}
}

func (r *RequestHandler) Close() {
	r.closeFunc.Close()
}
