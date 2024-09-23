package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	clientIS "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	naClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	eventstoreConfig "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/config"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	pbRD "github.com/plgd-dev/hub/v2/resource-directory/pb"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	pbRD.UnimplementedResourceDirectoryServer

	resourceProjection  *Projection
	eventStore          eventstore.EventStore
	publicConfiguration PublicConfiguration
	ownerCache          *clientIS.OwnerCache
	closeFunc           fn.FuncList
	hubID               string
}

func (r *RequestHandler) Close() {
	r.closeFunc.Execute()
}

func AddHandler(ctx context.Context, svr *server.Server, config Config, publicConfiguration PublicConfiguration, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, goroutinePoolGo func(func()) error) error {
	handler, err := newRequestHandlerFromConfig(ctx, config, publicConfiguration, fileWatcher, logger, tracerProvider, goroutinePoolGo)
	if err != nil {
		return err
	}
	svr.AddCloseFunc(handler.Close)
	pb.RegisterGrpcGatewayServer(svr.Server, handler)
	pbRD.RegisterResourceDirectoryServer(svr.Server, handler)
	return nil
}

// Register registers the handler instance with a gRPC server.
func Register(server *grpc.Server, handler *RequestHandler) {
	pb.RegisterGrpcGatewayServer(server, handler)
}

func newIdentityStoreClient(config IdentityStoreConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	var closeIsClient fn.FuncList

	isConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to identity-store: %w", err)
	}

	closeIsClient.AddFunc(func() {
		if err := isConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to identity-store: %w", err)
		}
	})

	isClient := pbIS.NewIdentityStoreClient(isConn.GRPC())
	return isClient, closeIsClient.ToFunction(), nil
}

func createEventStore(ctx context.Context, config eventstoreConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (eventstore.EventStore, error) {
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, tracerProvider, cqldb.WithUnmarshaler(utils.Unmarshal), cqldb.WithMarshaler(utils.Marshal))
		if err != nil {
			return nil, fmt.Errorf("cqldb: %w", err)
		}
		return s, nil
	}
	return nil, fmt.Errorf("invalid eventstore use('%v')", config.Use)
}

func newRequestHandlerFromConfig(ctx context.Context, config Config, publicConfiguration PublicConfiguration, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc fn.FuncList
	if publicConfiguration.CAPool != "" {
		content, err := publicConfiguration.CAPool.Read()
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", publicConfiguration.CAPool, err)
		}
		publicConfiguration.cloudCertificateAuthorities = string(content)
	}

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	closeFunc.AddFunc(closeIsClient)

	eventstore, err := createEventStore(ctx, config.Clients.Eventstore.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource eventstore %w", err)
	}
	closeFunc.AddFunc(func() {
		if errC := eventstore.Close(ctx); errC != nil {
			logger.Errorf("error occurs during close connection to eventstore: %w", errC)
		}
	})

	natsClient, err := naClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	closeFunc.AddFunc(natsClient.Close)

	resourceSubscriber, err := subscriber.New(natsClient.GetConn(),
		config.Clients.Eventbus.NATS.PendingLimits, config.Clients.Eventbus.NATS.LeadResourceType.IsEnabled(),
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

	ownerCache := clientIS.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim,
		config.APIs.GRPC.OwnerCacheExpiration,
		natsClient.GetConn(),
		isClient, func(err error) {
			log.Errorf("ownerCache error: %w", err)
		},
	)

	closeFunc.AddFunc(ownerCache.Close)

	h := NewRequestHandler(
		config.HubID,
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
	hubID string,
	resourceProjection *Projection,
	eventstore eventstore.EventStore,
	publicConfiguration PublicConfiguration,
	ownerCache *clientIS.OwnerCache,
	closeFunc fn.FuncList,
) *RequestHandler {
	return &RequestHandler{
		hubID:               hubID,
		resourceProjection:  resourceProjection,
		eventStore:          eventstore,
		publicConfiguration: publicConfiguration,
		ownerCache:          ownerCache,
		closeFunc:           closeFunc,
	}
}

func NewEventStoreModelFactory() func(context.Context, string, string) (eventstore.Model, error) {
	return func(_ context.Context, deviceID, resourceID string) (eventstore.Model, error) {
		switch resourceID {
		case commands.MakeLinksResourceUUID(deviceID).String():
			return NewResourceLinksProjection(deviceID), nil
		case commands.MakeStatusResourceUUID(deviceID).String():
			return NewDeviceMetadataProjection(deviceID), nil
		}
		return NewResourceProjection(), nil
	}
}
