package service

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"go.uber.org/zap"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	authServiceClient pbAS.AuthorizationServiceClient

	resourceProjection            *Projection
	subscriptions                 *Subscriptions
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	publicConfiguration           PublicConfiguration
	userDevicesManager            *clientAS.UserDevicesManager
	closeFunc                     closeFunc
}

// AddCloseFunc adds a function to be called by the Close method.
func (s *RequestHandler) AddCloseFunc(f func()) {
	s.closeFunc = append(s.closeFunc, f)
}

func (s *RequestHandler) Close() {
	s.closeFunc.Close()
}

func AddHandler(ctx context.Context, svr *server.Server, config ClientsConfig, publicConfiguration PublicConfiguration, logger *zap.Logger, goroutinePoolGo func(func()) error) error {
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

func NewRequestHandlerFromConfig(ctx context.Context, config ClientsConfig, publicConfiguration PublicConfiguration, logger *zap.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc closeFunc
	if publicConfiguration.CAPool != "" {
		content, err := ioutil.ReadFile(publicConfiguration.CAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", publicConfiguration.CAPool, err)
		}
		publicConfiguration.cloudCertificateAuthorities = string(content)
	}

	oauthMgr, err := manager.New(config.AuthServer.OAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	closeFunc = append(closeFunc, oauthMgr.Close)

	asConn, err := client.New(config.AuthServer.Connection, logger, grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := asConn.Close()
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to authorization server: %w", err)
		}
	})
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn.GRPC())

	eventstore, err := mongodb.New(ctx, config.Eventstore.Connection.MongoDB, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := eventstore.Close(ctx)
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to mongodb: %w", err)
		}
	})

	resourceSubscriber, err := subscriber.New(config.Eventbus.NATS, logger, subscriber.WithGoPool(goroutinePoolGo))
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}
	closeFunc = append(closeFunc, resourceSubscriber.Close)

	subscriptions := NewSubscriptions()
	userDevicesManager := clientAS.NewUserDevicesManager(subscriptions.UserDevicesChanged, authServiceClient, config.AuthServer.PullFrequency, config.AuthServer.CacheExpiration, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving devices: %v", err) })
	subscriptions.userDevicesManager = userDevicesManager
	closeFunc = append(closeFunc, userDevicesManager.Close)

	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()
	createNotificationContainer := notification.NewCreateNotificationContainer()
	projUUID, err := uuid.NewV4()
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create uuid for projection %w", err)
	}
	resourceProjection, err := NewProjection(ctx, projUUID.String(), eventstore, resourceSubscriber, NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), config.Eventstore.ProjectionCacheExpiration)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	h := NewRequestHandler(
		authServiceClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
		deleteNotificationContainer,
		publicConfiguration,
		userDevicesManager,
		closeFunc,
	)
	return h, nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	authServiceClient pbAS.AuthorizationServiceClient,
	resourceProjection *Projection,
	subscriptions *Subscriptions,
	updateNotificationContainer *notification.UpdateNotificationContainer,
	retrieveNotificationContainer *notification.RetrieveNotificationContainer,
	deleteNotificationContainer *notification.DeleteNotificationContainer,
	publicConfiguration PublicConfiguration,
	userDevicesManager *clientAS.UserDevicesManager,
	closeFunc []func(),
) *RequestHandler {
	return &RequestHandler{
		authServiceClient:             authServiceClient,
		resourceProjection:            resourceProjection,
		subscriptions:                 subscriptions,
		updateNotificationContainer:   updateNotificationContainer,
		retrieveNotificationContainer: retrieveNotificationContainer,
		deleteNotificationContainer:   deleteNotificationContainer,
		publicConfiguration:           publicConfiguration,
		userDevicesManager:            userDevicesManager,
		closeFunc:                     closeFunc,
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
