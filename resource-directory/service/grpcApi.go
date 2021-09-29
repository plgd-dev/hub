package service

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	clientIS "github.com/plgd-dev/cloud/identity/client"
	pbIS "github.com/plgd-dev/cloud/identity/pb"
	"github.com/plgd-dev/cloud/pkg/fn"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	naClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer

	resourceProjection  *Projection
	eventStore          eventstore.EventStore
	publicConfiguration PublicConfiguration
	ownerCache          *clientIS.OwnerCache
	closeFunc           fn.FuncList
}

func (s *RequestHandler) Close() {
	s.closeFunc.Execute()
}

func AddHandler(ctx context.Context, svr *server.Server, config Config, publicConfiguration PublicConfiguration, logger log.Logger, goroutinePoolGo func(func()) error) error {
	handler, err := newRequestHandlerFromConfig(ctx, config, publicConfiguration, logger, goroutinePoolGo)
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

func newIdentityServiceClient(config IdentityServerConfig, logger log.Logger) (pbIS.IdentityServiceClient, func(), error) {
	var closeIsClient fn.FuncList

	isConn, err := client.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to identity server: %w", err)
	}

	closeIsClient.AddFunc(func() {
		if err := isConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to identity server: %w", err)
		}
	})

	isClient := pbIS.NewIdentityServiceClient(isConn.GRPC())
	return isClient, closeIsClient.ToFunction(), nil
}

func newRequestHandlerFromConfig(ctx context.Context, config Config, publicConfiguration PublicConfiguration, logger log.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc fn.FuncList
	if publicConfiguration.CAPool != "" {
		content, err := ioutil.ReadFile(publicConfiguration.CAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", publicConfiguration.CAPool, err)
		}
		publicConfiguration.cloudCertificateAuthorities = string(content)
	}

	isClient, closeIsClient, err := newIdentityServiceClient(config.Clients.IdentityServer, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create identity server client: %w", err)
	}
	closeFunc.AddFunc(closeIsClient)

	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}
	closeFunc.AddFunc(func() {
		if err := eventstore.Close(ctx); err != nil {
			logger.Errorf("error occurs during close connection to mongodb: %w", err)
		}
	})

	natsClient, err := naClient.New(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	closeFunc.AddFunc(natsClient.Close)

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

	mf := NewEventStoreModelFactory()
	projUUID, err := uuid.NewRandom()
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create uuid for projection %w", err)
	}
	resourceProjection, err := NewProjection(ctx, projUUID.String(), eventstore, resourceSubscriber, mf, config.Clients.Eventstore.ProjectionCacheExpiration)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	ownerCache := clientIS.NewOwnerCache("sub", config.APIs.GRPC.OwnerCacheExpiration, natsClient.GetConn(), isClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})

	h := NewRequestHandler(
		resourceProjection,
		eventstore,
		publicConfiguration,
		ownerCache,
		closeFunc,
	)
	return h, nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	resourceProjection *Projection,
	eventstore eventstore.EventStore,
	publicConfiguration PublicConfiguration,
	ownerCache *clientIS.OwnerCache,
	closeFunc fn.FuncList,
) *RequestHandler {
	return &RequestHandler{
		resourceProjection:  resourceProjection,
		eventStore:          eventstore,
		publicConfiguration: publicConfiguration,
		ownerCache:          ownerCache,
		closeFunc:           closeFunc,
	}
}

func NewEventStoreModelFactory() func(context.Context, string, string) (eventstore.Model, error) {
	return func(ctx context.Context, deviceID, resourceID string) (eventstore.Model, error) {
		switch resourceID {
		case commands.MakeLinksResourceUUID(deviceID):
			return NewResourceLinksProjection(), nil
		case commands.MakeStatusResourceUUID(deviceID):
			return NewDeviceMetadataProjection(), nil
		}
		return NewResourceProjection(), nil
	}
}
