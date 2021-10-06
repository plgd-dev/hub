package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	isClient "github.com/plgd-dev/cloud/v2/identity-store/client"
	pbIS "github.com/plgd-dev/cloud/v2/identity-store/pb"
	"github.com/plgd-dev/cloud/v2/pkg/fn"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	"github.com/plgd-dev/cloud/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/v2/pkg/net/grpc/server"
	raClient "github.com/plgd-dev/cloud/v2/resource-aggregate/client"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus"
	naClient "github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	idClient                pbIS.IdentityStoreClient
	resourceDirectoryClient pb.GrpcGatewayClient
	resourceAggregateClient *raClient.Client
	resourceSubscriber      *subscriber.Subscriber
	ownerCache              *isClient.OwnerCache
	config                  Config
	closeFunc               func()
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

func newIdentityStoreClient(config IdentityStoreConfig, logger log.Logger) (pbIS.IdentityStoreClient, func(), error) {
	idConn, err := client.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to identity-store: %w", err)
	}
	closeIdConn := func() {
		if err := idConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to identity-store: %w", err)
		}
	}
	idClient := pbIS.NewIdentityStoreClient(idConn.GRPC())
	return idClient, closeIdConn, nil
}

func newResourceDirectoryClient(config GrpcServerConfig, logger log.Logger) (pb.GrpcGatewayClient, func(), error) {
	rdConn, err := client.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to resource-directory: %w", err)
	}
	closeRdConn := func() {
		if err := rdConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to resource-directory: %w", err)
		}
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn.GRPC())
	return resourceDirectoryClient, closeRdConn, nil
}

func newResourceAggregateClient(config GrpcServerConfig, resourceSubscriber eventbus.Subscriber, logger log.Logger) (*raClient.Client, func(), error) {
	raConn, err := client.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to resource-aggregate: %w", err)
	}
	closeRaConn := func() {
		if err := raConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to resource-aggregate: %w", err)
		}
	}
	raClient := raClient.New(raConn.GRPC(), resourceSubscriber)
	return raClient, closeRaConn, nil
}

func NewRequestHandlerFromConfig(ctx context.Context, config Config, logger log.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc fn.FuncList
	idClient, closeIdClient, err := newIdentityStoreClient(config.Clients.IdentityStore, logger)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	closeFunc.AddFunc(closeIdClient)

	natsClient, err := naClient.New(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	closeFunc.AddFunc(natsClient.Close)

	ownerCache := isClient.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration,
		natsClient.GetConn(), idClient, func(err error) {
			log.Errorf("error occurs during processing of event by ownerCache: %v", err)
		})
	closeFunc.AddFunc(ownerCache.Close)

	resourceDirectoryClient, closeResourceDirectoryClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, logger)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource-directory client: %w", err)
	}
	closeFunc.AddFunc(closeResourceDirectoryClient)

	resourceSubscriber, err := subscriber.New(natsClient.GetConn(),
		config.Clients.Eventbus.NATS.PendingLimits,
		logger,
		subscriber.WithGoPool(goroutinePoolGo),
		subscriber.WithUnmarshaler(utils.Unmarshal),
	)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}
	closeFunc.AddFunc(resourceSubscriber.Close)

	resourceAggregateClient, closeResourceAggregateClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, resourceSubscriber,
		logger)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource-aggregate client: %w", err)
	}
	closeFunc.AddFunc(closeResourceAggregateClient)

	return NewRequestHandler(
		idClient,
		resourceDirectoryClient,
		resourceAggregateClient,
		resourceSubscriber,
		ownerCache,
		config,
		closeFunc.ToFunction(),
	), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	idClient pbIS.IdentityStoreClient,
	resourceDirectoryClient pb.GrpcGatewayClient,
	resourceAggregateClient *raClient.Client,
	resourceSubscriber *subscriber.Subscriber,
	ownerCache *isClient.OwnerCache,
	config Config,
	closeFunc func(),
) *RequestHandler {
	return &RequestHandler{
		idClient:                idClient,
		resourceDirectoryClient: resourceDirectoryClient,
		resourceAggregateClient: resourceAggregateClient,
		resourceSubscriber:      resourceSubscriber,
		ownerCache:              ownerCache,
		config:                  config,
		closeFunc:               closeFunc,
	}
}

func (r *RequestHandler) Close() {
	r.closeFunc()
}
