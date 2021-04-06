package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"go.uber.org/zap"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	authServiceClient pbAS.AuthorizationServiceClient
	fqdn              string

	resourceProjection            *Projection
	subscriptions                 *Subscriptions
	seqNum                        uint64
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	timeoutForRequests            time.Duration
	clientConfiguration           pb.ClientConfigurationResponse
	userDevicesManager            *clientAS.UserDevicesManager
}

func AddHandler(ctx context.Context, svr *server.Server, config ClientsConfig, logger *zap.Logger, goroutinePoolGo func(func()) error) error {
	handler, err := NewRequestHandlerFromConfig(ctx, svr, config, logger, goroutinePoolGo)
	if err != nil {
		return err
	}
	pb.RegisterGrpcGatewayServer(svr.Server, handler)
	return nil
}

// Register registers the handler instance with a gRPC server.
func Register(server *grpc.Server, handler *RequestHandler) {
	pb.RegisterGrpcGatewayServer(server, handler)
}

func NewRequestHandlerFromConfig(ctx context.Context, server *server.Server, config ClientsConfig, logger *zap.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	if config.ClientConfiguration.CloudCAPool != "" {
		content, err := ioutil.ReadFile(config.ClientConfiguration.CloudCAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", config.ClientConfiguration.CloudCAPool, err)
		}
		config.ClientConfiguration.CloudCertificateAuthorities = string(content)
	}

	oauthMgr, err := manager.New(config.OAuthProvider.OAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	server.AddCloseFunc(oauthMgr.Close)

	asConn, err := client.New(config.AuthServer, logger, grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	server.AddCloseFunc(func() {
		err := asConn.Close()
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to authorization server: %w", err)
		}
	})
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn.GRPC())

	eventstore, err := mongodb.New(ctx, config.Eventstore.MongoDB, logger, goroutinePoolGo)
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}
	server.AddCloseFunc(func() {
		err := eventstore.Close(ctx)
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to mongodb: %w", err)
		}
	})

	resourceSubscriber, err := nats.NewSubscriberV2(config.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(clientTLS))
	if err != nil {
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}

	subscriptions := NewSubscriptions()
	userDevicesManager := clientAS.NewUserDevicesManager(subscriptions.UserDevicesChanged, authServiceClient, config.UserDevicesManagerTickFrequency, config.UserDevicesManagerExpiration, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving devices: %v", err) })
	subscriptions.userDevicesManager = userDevicesManager

	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()
	createNotificationContainer := notification.NewCreateNotificationContainer()
	projUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create uuid for projection %w", err)
	}
	resourceProjection, err := NewProjection(ctx, projUUID.String()+"."+config.FQDN, eventstore, resourceSubscriber, NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), config.ProjectionCacheExpiration)
	if err != nil {
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	h := NewRequestHandler(
		authServiceClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
		deleteNotificationContainer,
		config.TimeoutForRequests,
		config.ClientConfiguration.ClientConfigurationResponse,
		userDevicesManager,
		config.FQDN,
	)
	h.clientTLS = clientTLS
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
	timeoutForRequests time.Duration,
	closeFunc func(),
	clientConfiguration pb.ClientConfigurationResponse,
	userDevicesManager *clientAS.UserDevicesManager,
	fqdn string,
) *RequestHandler {
	return &RequestHandler{
		authServiceClient:             authServiceClient,
		resourceProjection:            resourceProjection,
		subscriptions:                 subscriptions,
		updateNotificationContainer:   updateNotificationContainer,
		retrieveNotificationContainer: retrieveNotificationContainer,
		deleteNotificationContainer:   deleteNotificationContainer,
		timeoutForRequests:            timeoutForRequests,
		closeFunc:                     closeFunc,
		clientConfiguration:           clientConfiguration,
		userDevicesManager:            userDevicesManager,
		fqdn:                          fqdn,
	}
}

func NewEventStoreModelFactory(subscriptions *Subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer, deleteNotificationContainer *notification.DeleteNotificationContainer, createNotificationContainer *notification.CreateNotificationContainer) func(context.Context, string, string) (eventstore.Model, error) {
	return func(ctx context.Context, deviceID, resourceID string) (eventstore.Model, error) {
		if commands.MakeLinksResourceUUID(deviceID) == resourceID {
			return NewResourceLinksProjection(subscriptions), nil
		}
		return NewResourceProjection(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), nil
	}
}

func logAndReturnError(err error) error {
	if errors.Is(err, io.EOF) {
		log.Debugf("%v", err)
		return err
	}
	if errors.Is(err, context.Canceled) {
		log.Debugf("%v", err)
		return err
	}
	log.Errorf("%v", err)
	return err
}

func (r *RequestHandler) SubscribeForEvents(srv pb.GrpcGateway_SubscribeForEventsServer) error {
	err := r.subscriptions.SubscribeForEvents(r.resourceProjection, srv)
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot subscribe for events: %v", err))
	}
	return nil
}
