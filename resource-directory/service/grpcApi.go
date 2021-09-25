package service

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/google/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/fn"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	naClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer

	resourceProjection  *Projection
	eventStore          eventstore.EventStore
	subscriptions       *Subscriptions
	publicConfiguration PublicConfiguration
	ownerCache          *clientAS.OwnerCache
	closeFunc           fn.FuncList
}

// AddCloseFunc adds a function to be called by the Close method.
func (s *RequestHandler) AddCloseFunc(f func()) {
	s.closeFunc.AddFunc(f)
}

func (s *RequestHandler) Close() {
	s.closeFunc.Execute()
}

func AddHandler(ctx context.Context, svr *server.Server, config Config, publicConfiguration PublicConfiguration, logger log.Logger, goroutinePoolGo func(func()) error) error {
	handler, err := NewRequestHandlerFromConfig(ctx, config, publicConfiguration, logger, goroutinePoolGo)
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

func NewRequestHandlerFromConfig(ctx context.Context, config Config, publicConfiguration PublicConfiguration, logger log.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc fn.FuncList
	if publicConfiguration.CAPool != "" {
		content, err := ioutil.ReadFile(publicConfiguration.CAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", publicConfiguration.CAPool, err)
		}
		publicConfiguration.cloudCertificateAuthorities = string(content)
	}

	oauthMgr, err := manager.New(config.Clients.AuthServer.OAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	closeFunc.AddFunc(oauthMgr.Close)

	asConn, err := client.New(config.Clients.AuthServer.Connection, logger, grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	closeFunc.AddFunc(func() {
		if err := asConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to authorization server: %w", err)
		}
	})
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn.GRPC())

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

	subscriptions := NewSubscriptions()
	userDevicesManager := clientAS.NewUserDevicesManager(subscriptions.UserDevicesChanged, authServiceClient,
		config.Clients.AuthServer.PullFrequency, config.Clients.AuthServer.CacheExpiration,
		func(err error) { log.Errorf("grpc-gateway: error occurs during receiving devices: %v", err) })
	subscriptions.userDevicesManager = userDevicesManager
	closeFunc.AddFunc(userDevicesManager.Close)

	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()
	createNotificationContainer := notification.NewCreateNotificationContainer()
	mf := NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer)
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

	ownerCache := clientAS.NewOwnerCache("sub", config.APIs.GRPC.OwnerCacheExpiration, natsClient.GetConn(), authServiceClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})

	h := NewRequestHandler(
		resourceProjection,
		eventstore,
		subscriptions,
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
	subscriptions *Subscriptions,
	publicConfiguration PublicConfiguration,
	ownerCache *clientAS.OwnerCache,
	closeFunc fn.FuncList,
) *RequestHandler {
	return &RequestHandler{
		resourceProjection:  resourceProjection,
		eventStore:          eventstore,
		subscriptions:       subscriptions,
		publicConfiguration: publicConfiguration,
		ownerCache:          ownerCache,
		closeFunc:           closeFunc,
	}
}

func NewEventStoreModelFactory(subscriptions *Subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer, deleteNotificationContainer *notification.DeleteNotificationContainer, createNotificationContainer *notification.CreateNotificationContainer) func(context.Context, string, string) (eventstore.Model, error) {
	return func(ctx context.Context, deviceID, resourceID string) (eventstore.Model, error) {
		switch resourceID {
		case commands.MakeLinksResourceUUID(deviceID):
			return NewResourceLinksProjection(subscriptions), nil
		case commands.MakeStatusResourceUUID(deviceID):
			return NewDeviceMetadataProjection(subscriptions), nil
		}
		return NewResourceProjection(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), nil
	}
}

func (r *RequestHandler) SubscribeToEvents(srv pb.GrpcGateway_SubscribeToEventsServer) error {
	err := r.subscriptions.SubscribeToEvents(r.resourceProjection, srv)
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot subscribe to events: %v", err))
	}
	return nil
}
