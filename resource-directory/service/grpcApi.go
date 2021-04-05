package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"time"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"

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

func AddHandler(svr *kitNetGrpc.Server, config Config, logger *zap.Logger) error {
	handler, err := NewRequestHandlerFromConfig(config, logger)
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

func NewRequestHandlerFromConfig(config Config, logger *zap.Logger) (*RequestHandler, error) {
	svc := config.Service
	cli := config.Clients
	db := config.Database

	if cli.ClientConfiguration.CloudCAPool != "" {
		content, err := ioutil.ReadFile(cli.ClientConfiguration.CloudCAPool)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %v: %w", cli.ClientConfiguration.CloudCAPool, err)
		}
		cli.ClientConfiguration.CloudCertificateAuthorities = string(content)
	}

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err := config.Clients.OAuthProvider.TLSConfig.Validate()
	if err != nil {
		log.Errorf("failed to validate client tls config: %v", err)
	} else {
		oauthCertManager, err := client.New(config.Clients.OAuthProvider.TLSConfig, logger)
		if err != nil {
			log.Errorf("cannot create oauth client cert manager %v", err)
		} else {
			oauthTLSConfig = oauthCertManager.GetTLSConfig()
		}
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(cli.OAuthProvider.OAuth, oauthTLSConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}

	asCertManager, err := client.New(cli.Authorization.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create as grpc client cert manager %w", err)
	}
	asConn, err := grpc.Dial(cli.Authorization.Addr, grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())), grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn)

	pool, err := ants.NewPool(svc.Grpc.Capabilities.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}

	mongoCertManager, err := client.New(db.MongoDB.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create mongodb client cert manager %w", err)
	}
	ctx := context.Background()
	resourceEventStore, err := mongodb.NewEventStore(ctx, db.MongoDB, pool.Submit, mongodb.WithTLS(mongoCertManager.GetTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}

	natsCertManager, err := client.New(cli.Nats.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create nats client cert manager %w", err)
	}
	resourceSubscriber, err := nats.NewSubscriber(cli.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(natsCertManager.GetTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}

	subscriptions := NewSubscriptions()
	userDevicesManager := clientAS.NewUserDevicesManager(subscriptions.UserDevicesChanged, authServiceClient, svc.Grpc.Capabilities.UserDevicesManagerTickFrequency, svc.Grpc.Capabilities.UserDevicesManagerExpiration, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving devices: %v", err) })
	subscriptions.userDevicesManager = userDevicesManager

	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()
	createNotificationContainer := notification.NewCreateNotificationContainer()
	projUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create uuid for projection %w", err)
	}
	resourceProjection, err := NewProjection(ctx, projUUID.String()+"."+svc.Grpc.FQDN, resourceEventStore, resourceSubscriber, NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), svc.Grpc.Capabilities.ProjectionCacheExpiration)
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

		if oauthCertManager != nil { oauthCertManager.Close() }
		asCertManager.Close()
		//raCertManager.Close()
		mongoCertManager.Close()
		natsCertManager.Close()
	}

	h := NewRequestHandler(
		authServiceClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
		deleteNotificationContainer,
		svc.Grpc.Capabilities.TimeoutForRequests,
		closeFunc,
		cli.ClientConfiguration.ClientConfigurationResponse,
		userDevicesManager,
		svc.Grpc.FQDN,
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
