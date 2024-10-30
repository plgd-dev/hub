package service

import (
	"fmt"

	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	isClient "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	naClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	idClient                   pbIS.IdentityStoreClient
	resourceDirectoryClient    pb.GrpcGatewayClient
	resourceAggregateClient    *raClient.Client
	certificateAuthorityClient pbCA.CertificateAuthorityClient
	resourceSubscriber         *subscriber.Subscriber
	ownerCache                 *isClient.OwnerCache
	subscriptionsCache         *subscription.SubscriptionsCache
	logger                     log.Logger
	config                     Config
	closeFunc                  func()
}

func addHandler(svr *server.Server, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, goroutinePoolGo func(func()) error) error {
	handler, err := newRequestHandlerFromConfig(config, fileWatcher, logger, tracerProvider, goroutinePoolGo)
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

func newIdentityStoreClient(config IdentityStoreConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	idConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newResourceDirectoryClient(config GrpcServerConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pb.GrpcGatewayClient, func(), error) {
	rdConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newCertificateAuthorityClient(config GrpcServerConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbCA.CertificateAuthorityClient, func(), error) {
	caConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to certificate-authority: %w", err)
	}
	closeCaConn := func() {
		if err := caConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to certificate-authority: %w", err)
		}
	}
	certificateAuthorityClient := pbCA.NewCertificateAuthorityClient(caConn.GRPC())
	return certificateAuthorityClient, closeCaConn, nil
}

func newResourceAggregateClient(config GrpcServerConfig, resourceSubscriber eventbus.Subscriber, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*raClient.Client, func(), error) {
	raConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newRequestHandlerFromConfig(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc fn.FuncList
	idClient, closeIdClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	closeFunc.AddFunc(closeIdClient)

	natsClient, err := naClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	closeFunc.AddFunc(natsClient.Close)

	ownerCache := isClient.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration,
		natsClient.GetConn(), idClient, func(err error) {
			logger.Errorf("error occurs during processing of event by ownerCache: %v", err)
		})
	closeFunc.AddFunc(ownerCache.Close)

	resourceDirectoryClient, closeResourceDirectoryClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource-directory client: %w", err)
	}
	closeFunc.AddFunc(closeResourceDirectoryClient)

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

	resourceAggregateClient, closeResourceAggregateClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, resourceSubscriber,
		fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create resource-aggregate client: %w", err)
	}
	closeFunc.AddFunc(closeResourceAggregateClient)

	certificateAuthorityClient, closeCertificateAuthorityClient, err := newCertificateAuthorityClient(config.Clients.CertificateAuthority, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeFunc.Execute()
		return nil, fmt.Errorf("cannot create certificate-authority client: %w", err)
	}
	closeFunc.AddFunc(closeCertificateAuthorityClient)

	subscriptionsCache := subscription.NewSubscriptionsCache(natsClient.GetConn(), func(err error) {
		logger.Errorf("error occurs during processing of event by subscriptionCache: %v", err)
	})

	return &RequestHandler{
		idClient:                   idClient,
		resourceDirectoryClient:    resourceDirectoryClient,
		resourceAggregateClient:    resourceAggregateClient,
		certificateAuthorityClient: certificateAuthorityClient,
		resourceSubscriber:         resourceSubscriber,
		ownerCache:                 ownerCache,
		subscriptionsCache:         subscriptionsCache,
		config:                     config,
		closeFunc:                  closeFunc.ToFunction(),
		logger:                     logger,
	}, nil
}

func (r *RequestHandler) Close() {
	r.closeFunc()
}
