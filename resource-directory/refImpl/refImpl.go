package refImpl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-ocf/kit/security/certManager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbAS "github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/eventstore/mongodb"
	pbDD "github.com/go-ocf/ocf-cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/ocf-cloud/resource-directory/service"
	"github.com/panjf2000/ants"
)

type Config struct {
	Service           service.Config
	GoRoutinePoolSize int                `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	CacheExpiration   time.Duration      `envconfig:"CACHE_EXPIRATION" default:"30s"`
	Nats              nats.Config        `envconfig:"NATS"`
	MongoDB           mongodb.Config     `envconfig:"MONGODB"`
	Log               log.Config         `envconfig:"LOG"`
	Listen            certManager.Config `envconfig:"LISTEN"`
	Dial              certManager.Config `envconfig:"DIAL"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

type RefImpl struct {
	eventstore        *mongodb.EventStore
	handle            *service.RequestHandler
	subscriber        *nats.Subscriber
	clientCertManager certManager.CertManager
	serverCertManager certManager.CertManager
	server            *kitNetGrpc.Server
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)

	log.Info(config.String())

	impl, err := NewRequestHandlerFromConfig(config)
	if err != nil {
		return nil, err
	}

	serverTLSConfig := impl.serverCertManager.GetServerTLSConfig()

	svr, err := kitNetGrpc.NewServer(config.Service.Addr, grpc.Creds(credentials.NewTLS(&serverTLSConfig)))
	if err != nil {
		return nil, err
	}
	pbRS.RegisterResourceShadowServer(svr.Server, impl.handle)
	pbRD.RegisterResourceDirectoryServer(svr.Server, impl.handle)
	pbDD.RegisterDeviceDirectoryServer(svr.Server, impl.handle)
	impl.server = svr
	return impl, nil
}

func (r *RefImpl) Serve() error {
	return r.server.Serve()
}

func (r *RefImpl) Shutdown() {
	r.server.Stop()
	r.eventstore.Close(context.Background())
	r.subscriber.Close()
	r.clientCertManager.Close()
	r.serverCertManager.Close()
}

// NewRequestHandlerFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRequestHandlerFromConfig(config Config) (*RefImpl, error) {

	clientCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create client cert manager %v", err)
	}
	serverCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %v", err)
	}

	svc := config.Service
	clientTLSConfig := clientCertManager.GetClientTLSConfig()

	authServiceConn, err := grpc.Dial(svc.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLSConfig)))
	authServiceClient := pbAS.NewAuthorizationServiceClient(authServiceConn)

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %v", err)
	}

	eventstore, err := mongodb.NewEventStore(config.MongoDB, pool.Submit, mongodb.WithTLS(&clientTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %v", err)
	}

	subscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("resource-directory: error occurs during receiving event: %v", err) }, nats.WithTLS(&clientTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource nats subscriber %v", err)
	}

	ctx := context.Background()

	resourceProjection, err := service.NewProjection(ctx, svc.FQDN, eventstore, subscriber, config.CacheExpiration)
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %v", err)
	}

	return &RefImpl{
		eventstore:        eventstore,
		handle:            service.NewRequestHandler(authServiceClient, resourceProjection),
		subscriber:        subscriber,
		clientCertManager: clientCertManager,
		serverCertManager: serverCertManager,
	}, nil
}
