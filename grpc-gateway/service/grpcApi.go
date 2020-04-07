package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"time"

	clientAS "github.com/go-ocf/cloud/authorization/client"
	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/gofrs/uuid"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/go-ocf/kit/security/oauth/manager"
	"github.com/panjf2000/ants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	deviceDirectoryClient   pbDD.DeviceDirectoryClient
	resourceDirectoryClient pbRD.ResourceDirectoryClient
	resourceShadowClient    pbRS.ResourceShadowClient
	authServiceClient       pbAS.AuthorizationServiceClient
	resourceAggregateClient pbRA.ResourceAggregateClient

	resourceProjection            *projectionRA.Projection
	subscriptions                 *subscriptions
	seqNum                        uint64
	clientTLS                     tls.Config
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	timeoutForRequests            time.Duration
	closeFunc                     func()
}

type HandlerConfig struct {
	Mongo   mongodb.Config
	Nats    nats.Config
	Service Config

	GoRoutinePoolSize               int           `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	UserDevicesManagerTickFrequency time.Duration `envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration `envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
}

func AddHandler(svr *kitNetGrpc.Server, config HandlerConfig, clientTLS tls.Config) error {
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

func NewRequestHandlerFromConfig(config HandlerConfig, clientTLS tls.Config) (*RequestHandler, error) {
	svc := config.Service

	rdConn, err := grpc.Dial(svc.ResourceDirectoryAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLS)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource directory: %w", err)
	}
	deviceDirectoryClient := pbDD.NewDeviceDirectoryClient(rdConn)
	resourceDirectoryClient := pbRD.NewResourceDirectoryClient(rdConn)
	resourceShadowClient := pbRS.NewResourceShadowClient(rdConn)

	oauthMgr, err := manager.NewManagerFromConfiguration(svc.OAuth, &clientTLS)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}

	asConn, err := grpc.Dial(svc.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLS)), grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to authorization server: %w", err)
	}
	authServiceClient := pbAS.NewAuthorizationServiceClient(asConn)

	raConn, err := grpc.Dial(svc.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLS)))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceAggregateClient := pbRA.NewResourceAggregateClient(raConn)

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}

	resourceEventStore, err := mongodb.NewEventStore(config.Mongo, pool.Submit, mongodb.WithTLS(&clientTLS))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %w", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("grpc-gateway: error occurs during receiving event: %v", err) }, nats.WithTLS(&clientTLS))
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
	resourceProjection, err := projectionRA.NewProjection(ctx, projUUID.String()+"."+svc.FQDN, resourceEventStore, resourceSubscriber, NewResourceCtx(subscriptions, updateNotificationContainer, retrieveNotificationContainer))
	if err != nil {
		return nil, fmt.Errorf("cannot create projection over resource aggregate events: %w", err)
	}

	closeFunc := func() {
		resourceSubscriber.Close()
		resourceEventStore.Close(context.Background())
		userDevicesManager.Close()
		pool.Release()
		raConn.Close()
		rdConn.Close()
		asConn.Close()
		oauthMgr.Close()
	}

	h := NewRequestHandler(
		authServiceClient,
		deviceDirectoryClient,
		resourceDirectoryClient,
		resourceShadowClient,
		resourceAggregateClient,
		resourceProjection,
		subscriptions,
		updateNotificationContainer,
		retrieveNotificationContainer,
		svc.TimeoutForRequests,
		closeFunc)
	h.clientTLS = clientTLS
	return h, nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	authServiceClient pbAS.AuthorizationServiceClient,
	deviceDirectoryClient pbDD.DeviceDirectoryClient,
	resourceDirectoryClient pbRD.ResourceDirectoryClient,
	resourceShadowClient pbRS.ResourceShadowClient,
	resourceAggregateClient pbRA.ResourceAggregateClient,
	resourceProjection *projectionRA.Projection,
	subscriptions *subscriptions,
	updateNotificationContainer *notification.UpdateNotificationContainer,
	retrieveNotificationContainer *notification.RetrieveNotificationContainer,
	timeoutForRequests time.Duration,
	closeFunc func(),
) *RequestHandler {
	return &RequestHandler{
		authServiceClient:             authServiceClient,
		deviceDirectoryClient:         deviceDirectoryClient,
		resourceDirectoryClient:       resourceDirectoryClient,
		resourceShadowClient:          resourceShadowClient,
		resourceProjection:            resourceProjection,
		subscriptions:                 subscriptions,
		resourceAggregateClient:       resourceAggregateClient,
		updateNotificationContainer:   updateNotificationContainer,
		retrieveNotificationContainer: retrieveNotificationContainer,
		timeoutForRequests:            timeoutForRequests,
		closeFunc:                     closeFunc,
	}
}

func grpcStatus2ddStatus(in pb.GetDevicesRequest_Status) pbDD.Status {
	if in == pb.GetDevicesRequest_ONLINE {
		return pbDD.Status_ONLINE
	}
	return pbDD.Status_OFFLINE
}

func ddLocalizedString2grpcLocalizedString(in *pbDD.LocalizedString) *pb.LocalizedString {
	return &pb.LocalizedString{
		Language: in.Language,
		Value:    in.Value,
	}
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}

func (r *RequestHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	accessToken, err := grpc_auth.AuthFromMD(srv.Context(), "bearer")
	if err != nil {
		return logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot get devices: %v", err))
	}
	statusFilter := make([]pbDD.Status, 0, 2)
	for _, s := range req.StatusFilter {
		statusFilter = append(statusFilter, grpcStatus2ddStatus(s))
	}
	ddReq := pbDD.GetDevicesRequest{
		TypeFilter:      req.TypeFilter,
		StatusFilter:    statusFilter,
		DeviceIdsFilter: req.DeviceIdsFilter,
	}
	getDevicesClient, err := r.deviceDirectoryClient.GetDevices(kitNetGrpc.CtxWithToken(srv.Context(), accessToken), &ddReq)
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Unauthenticated, "cannot get devices: %v", err))
	}
	defer getDevicesClient.CloseSend()

	for {
		ddDev, err := getDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get devices: %v", err))
		}

		manufacturerName := make([]*pb.LocalizedString, 0, 16)
		for _, l := range ddDev.GetResource().GetManufacturerName() {
			manufacturerName = append(manufacturerName, ddLocalizedString2grpcLocalizedString(l))
		}

		devResp := pb.Device{
			Id:               ddDev.GetId(),
			Types:            ddDev.GetResource().GetResourceTypes(),
			Name:             ddDev.GetResource().GetName(),
			IsOnline:         ddDev.GetIsOnline(),
			ManufacturerName: manufacturerName,
			ModelNumber:      ddDev.GetResource().GetModelNumber(),
		}
		err = srv.Send(&devResp)
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get devices: %v", err))
		}
	}

	return nil
}

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	accessToken, err := grpc_auth.AuthFromMD(srv.Context(), "bearer")
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.NotFound, "cannot get resource links: %v", err))
	}

	rdReq := pbRD.GetResourceLinksRequest{
		TypeFilter:      req.TypeFilter,
		DeviceIdsFilter: req.DeviceIdsFilter,
	}
	getResourceLinksClient, err := r.resourceDirectoryClient.GetResourceLinks(kitNetGrpc.CtxWithToken(srv.Context(), accessToken), &rdReq)
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get resource links: %v", err))
	}
	defer getResourceLinksClient.CloseSend()
	for {
		rdRes, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get resource links: %v", err))
		}
		resResp := makeResourceLink(rdRes.Resource)
		err = srv.Send(&resResp)
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get resource links: %v", err))
		}
	}

	return nil
}

func GrpcResourceID2ResourceID(resourceId *pb.ResourceId) string {
	return cqrsRA.MakeResourceId(resourceId.DeviceId, resourceId.ResourceLinkHref)
}

func (r *RequestHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	accessToken, err := grpc_auth.AuthFromMD(srv.Context(), "bearer")
	if err != nil {
		return logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot retrieve resources values: %v", err))
	}
	resourceIds := make([]string, 0, 16)
	for _, r := range req.ResourceIdsFilter {
		resourceIds = append(resourceIds, GrpcResourceID2ResourceID(r))
	}
	rdReq := pbRS.RetrieveResourcesValuesRequest{
		TypeFilter:        req.TypeFilter,
		DeviceIdsFilter:   req.DeviceIdsFilter,
		ResourceIdsFilter: resourceIds,
	}
	retrieveResourcesValuesClient, err := r.resourceShadowClient.RetrieveResourcesValues(kitNetGrpc.CtxWithToken(srv.Context(), accessToken), &rdReq)
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resources values: %v", err))
	}
	defer retrieveResourcesValuesClient.CloseSend()
	for {
		rdRes, err := retrieveResourcesValuesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resources values: %v", err))
		}
		if rdRes.Content == nil {
			continue
		}
		resResp := pb.ResourceValue{
			ResourceId: &pb.ResourceId{
				DeviceId:         rdRes.DeviceId,
				ResourceLinkHref: rdRes.Href,
			},
			Content: &pb.Content{
				Data:        rdRes.Content.Data,
				ContentType: rdRes.Content.ContentType,
			},
			Types: rdRes.Types,
		}
		err = srv.Send(&resResp)
		if err != nil {
			return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resources values: %v", err))
		}
	}
	return nil
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

func (r *RequestHandler) GetClientTLSConfig() tls.Config {
	return r.clientTLS
}
