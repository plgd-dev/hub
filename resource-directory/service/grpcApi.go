package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	clientAS "github.com/go-ocf/cloud/authorization/client"
	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"

	"github.com/go-ocf/kit/security/oauth/manager"
	"github.com/panjf2000/ants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	authServiceClient       pbAS.AuthorizationServiceClient
	resourceAggregateClient pbRA.ResourceAggregateClient
	fqdn                    string

	resourceProjection            *Projection
	subscriptions                 *subscriptions
	seqNum                        uint64
	clientTLS                     *tls.Config
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	timeoutForRequests            time.Duration
	closeFunc                     func()
	clientConfiguration           pb.ClientConfigurationResponse
	userDevicesManager            *clientAS.UserDevicesManager
}

type HandlerConfig struct {
	Mongo   mongodb.Config
	Nats    nats.Config
	Service Config

	GoRoutinePoolSize               int           `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	UserDevicesManagerTickFrequency time.Duration `envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration `envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
}

func AddHandler(svr *kitNetGrpc.Server, config HandlerConfig, clientTLS *tls.Config) error {
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

	raConn, err := grpc.Dial(svc.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceAggregateClient := pbRA.NewResourceAggregateClient(raConn)

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}

	resourceEventStore, err := mongodb.NewEventStore(config.Mongo, pool.Submit, mongodb.WithTLS(clientTLS))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving event: %v", err) }, nats.WithTLS(clientTLS))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource nats subscriber %w", err)
	}

	ctx := context.Background()

	subscriptions := NewSubscriptions()
	userDevicesManager := clientAS.NewUserDevicesManager(subscriptions.UserDevicesChanged, authServiceClient, config.UserDevicesManagerTickFrequency, config.UserDevicesManagerExpiration, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving devices: %v", err) })
	subscriptions.userDevicesManager = userDevicesManager

	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	projUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create uuid for projection %w", err)
	}
	resourceProjection, err := NewProjection(ctx, projUUID.String()+"."+svc.FQDN, resourceEventStore, resourceSubscriber, NewResourceCtx(subscriptions, updateNotificationContainer, retrieveNotificationContainer), time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	closeFunc := func() {
		resourceSubscriber.Close()
		resourceEventStore.Close(context.Background())
		userDevicesManager.Close()
		pool.Release()
		raConn.Close()
		asConn.Close()
		oauthMgr.Close()
	}

	h := NewRequestHandler(
		authServiceClient,
		resourceAggregateClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
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
	resourceAggregateClient pbRA.ResourceAggregateClient,
	resourceProjection *Projection,
	subscriptions *subscriptions,
	updateNotificationContainer *notification.UpdateNotificationContainer,
	retrieveNotificationContainer *notification.RetrieveNotificationContainer,
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
		resourceAggregateClient:       resourceAggregateClient,
		updateNotificationContainer:   updateNotificationContainer,
		retrieveNotificationContainer: retrieveNotificationContainer,
		timeoutForRequests:            timeoutForRequests,
		closeFunc:                     closeFunc,
		clientConfiguration:           clientConfiguration,
		userDevicesManager:            userDevicesManager,
		fqdn:                          fqdn,
	}
}

func logAndReturnError(err error) error {
	if errors.As(err, &io.EOF) {
		log.Debugf("%v", err)
		return err
	}
	if errors.As(err, &context.Canceled) {
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
