package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"github.com/plgd-dev/kit/log"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	authServiceClient pbAS.AuthorizationServiceClient
	fqdn              string

	resourceProjection            *Projection
	subscriptions                 *Subscriptions
	seqNum                        uint64
	clientTLS                     *tls.Config
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	timeoutForRequests            time.Duration
	closeFunc                     func()
	clientConfiguration           pb.ClientConfigurationResponse
	userDevicesManager            *clientAS.UserDevicesManager
}

type HandlerConfig struct {
	MongoDB mongodb.Config `envconfig:"MONGO"`
	Nats    nats.Config
	Service Config

	GoRoutinePoolSize               int           `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	UserDevicesManagerTickFrequency time.Duration `envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration `envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
}

func AddHandler(svr *server.Server, config HandlerConfig, clientTLS *tls.Config) error {
	handler, err := NewRequestHandlerFromConfig(config, clientTLS)
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

func NewRequestHandlerFromConfig(config HandlerConfig, clientTLS *tls.Config) (*RequestHandler, error) {
	svc := config.Service

	if svc.ClientConfiguration.CloudCAPool != "" {
		content, err := ioutil.ReadFile(svc.ClientConfiguration.CloudCAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", svc.ClientConfiguration.CloudCAPool, err)
		}
		svc.ClientConfiguration.CloudCertificateAuthorities = string(content)
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(svc.OAuth, clientTLS)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}

	asConn, err := grpc.Dial(svc.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)), grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn)

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}

	ctx := context.Background()
	resourceEventStore, err := mongodb.NewEventStore(ctx, config.MongoDB, pool.Submit, mongodb.WithTLS(clientTLS))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(clientTLS))
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
	resourceProjection, err := NewProjection(ctx, projUUID.String()+"."+svc.FQDN, resourceEventStore, resourceSubscriber, NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), svc.ProjectionCacheExpiration)
	if err != nil {
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	closeFunc := func() {
		resourceSubscriber.Close()
		resourceEventStore.Close(context.Background())
		userDevicesManager.Close()
		pool.Release()
		asConn.Close()
		oauthMgr.Close()
	}

	h := NewRequestHandler(
		authServiceClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
		deleteNotificationContainer,
		svc.TimeoutForRequests,
		closeFunc,
		svc.ClientConfiguration.ClientConfigurationResponse,
		userDevicesManager,
		svc.FQDN,
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

func (r *RequestHandler) Close() {
	r.closeFunc()
}

func (r *RequestHandler) GetClientTLSConfig() *tls.Config {
	return r.clientTLS
}
