package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/plgd-dev/hub/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/hub/cloud2cloud-connector/uri"
	pbGRPC "github.com/plgd-dev/hub/grpc-gateway/pb"
	pbIS "github.com/plgd-dev/hub/identity-store/pb"
	"github.com/plgd-dev/hub/pkg/fn"
	"github.com/plgd-dev/hub/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/pkg/mongodb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/pkg/net/grpc/client"
	kitNetHttp "github.com/plgd-dev/hub/pkg/net/http"
	"github.com/plgd-dev/hub/pkg/net/listener"
	cmClient "github.com/plgd-dev/hub/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/pkg/security/oauth2"
	natsClient "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	raService "github.com/plgd-dev/hub/resource-aggregate/service"
)

// Server handles HTTP request
type Server struct {
	server   *http.Server
	listener *listener.Server
	doneWg   *sync.WaitGroup
	cancel   context.CancelFunc
}

func toValidator(c oauth2.Config) validator.Config {
	return validator.Config{
		Authority: c.Authority,
		Audience:  c.Audience,
		HTTP:      c.HTTP,
	}
}

func newAuthInterceptor(ctx context.Context, config validator.Config, logger log.Logger) (kitNetHttp.Interceptor, func(), error) {
	var fl fn.FuncList

	validator, err := validator.New(ctx, config, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create validator: %w", err)
	}
	fl.AddFunc(validator.Close)

	var authRules = map[string][]kitNetHttp.AuthArgs{
		http.MethodGet: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			},
		},
		http.MethodPost: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			},
		},
		http.MethodDelete: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			},
		},
		http.MethodPut: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			},
		},
	}

	whiteList := []kitNetHttp.RequestMatcher{
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.OAuthCallback)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.OAuthCallback)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.Events)),
		},
	}
	auth := kitNetHttp.NewInterceptorWithValidator(validator, authRules, whiteList...)

	return auth, fl.ToFunction(), nil
}

func newIdentityStoreClient(config IdentityStoreConfig, logger log.Logger) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := grpcClient.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to identity-store: %w", err)
	}
	closeIsConn := func() {
		if err := isConn.Close(); err != nil && !kitNetGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during close connection to identity-store: %v", err)
		}
	}
	return pbIS.NewIdentityStoreClient(isConn.GRPC()), closeIsConn, nil
}

func newSubscriber(config natsClient.Config, logger log.Logger) (*subscriber.Subscriber, func(), error) {
	var fl fn.FuncList
	nats, err := natsClient.New(config, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	fl.AddFunc(nats.Close)

	sub, err := subscriber.New(nats.GetConn(),
		config.PendingLimits,
		logger, subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create subscriber: %v", err)
	}
	fl.AddFunc(sub.Close)

	return sub, fl.ToFunction(), nil
}

func newStore(ctx context.Context, config pkgMongo.Config, logger log.Logger) (*Store, func(), error) {
	var fl fn.FuncList
	certManager, err := cmClient.New(config.TLS, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	fl.AddFunc(certManager.Close)

	db, err := mongodb.NewStore(ctx, config, certManager.GetTLSConfig())
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create mongodb subscription store: %w", err)
	}
	fl.AddFunc(func() {
		if err := db.Close(ctx); err != nil {
			log.Errorf("failed to close subscription store: %w", err)
		}
	})

	store, err := NewStore(ctx, db)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cloud-to-cloud store: %w", err)
	}

	return store, fl.ToFunction(), nil
}

func newGrpcGatewayClient(config GrpcGatewayConfig, logger log.Logger) (pbGRPC.GrpcGatewayClient, func(), error) {
	var fl fn.FuncList
	grpcConn, err := grpcClient.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	fl.AddFunc(func() {
		if err := grpcConn.Close(); err != nil && !kitNetGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during closing of the connection to resource-aggregate: %w", err)
		}
	})
	grpcClient := pbGRPC.NewGrpcGatewayClient(grpcConn.GRPC())
	return grpcClient, fl.ToFunction(), nil
}

func newResourceAggregateClient(config ResourceAggregateConfig, logger log.Logger) (raService.ResourceAggregateClient, func(), error) {
	var fl fn.FuncList
	raConn, err := grpcClient.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	fl.AddFunc(func() {
		if err := raConn.Close(); err != nil && !kitNetGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during closing of the connection to resource-aggregate: %w", err)
		}
	})
	raClient := raService.NewResourceAggregateClient(raConn.GRPC())
	return raClient, fl.ToFunction(), nil
}

func newDevicesSubscription(ctx context.Context, config Config, raClient raService.ResourceAggregateClient, logger log.Logger) (*DevicesSubscription, func(), error) {
	var fl fn.FuncList

	grpcClient, closeGrpcClient, err := newGrpcGatewayClient(config.Clients.GrpcGateway, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create to grpc-gateway client: %w", err)
	}
	fl.AddFunc(closeGrpcClient)

	sub, closeSub, err := newSubscriber(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create subscriber: %w", err)
	}
	fl.AddFunc(closeSub)

	devicesSubscription := NewDevicesSubscription(ctx, grpcClient, raClient, sub, config.Clients.Subscription.HTTP.ReconnectInterval)
	return devicesSubscription, fl.ToFunction(), nil
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, logger log.Logger) (*Server, error) {
	listener, err := listener.New(config.APIs.HTTP.Connection, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}
	var cleanUp fn.FuncList
	cleanUp.AddFunc(func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("cannot create http server: %w", err)
		}
	})

	raClient, closeRaClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create to resource aggregate client: %w", err)
	}
	listener.AddCloseFunc(closeRaClient)

	ctx, cancel := context.WithCancel(ctx)
	cleanUp.AddFunc(cancel)
	devicesSubscription, closeDevSub, err := newDevicesSubscription(ctx, config, raClient, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create devices subscription subscriber: %w", err)
	}
	listener.AddCloseFunc(closeDevSub)

	taskProcessor := NewTaskProcessor(raClient, config.TaskProcessor.MaxParallel, config.TaskProcessor.CacheSize,
		config.TaskProcessor.Timeout, config.TaskProcessor.Delay)

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	listener.AddCloseFunc(closeIsClient)

	store, closeStore, err := newStore(ctx, config.Clients.Storage.MongoDB, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	listener.AddCloseFunc(closeStore)

	provider, err := oauth2.NewPlgdProvider(ctx, config.APIs.HTTP.Authorization.Config, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create device provider: %w", err)
	}
	listener.AddCloseFunc(provider.Close)

	subMgr := NewSubscriptionManager(config.APIs.HTTP.EventsURL, isClient, raClient, store, devicesSubscription,
		provider, taskProcessor.Trigger)

	requestHandler := NewRequestHandler(config.APIs.HTTP.Authorization.OwnerClaim, provider, subMgr, store, taskProcessor.Trigger)

	auth, closeAuth, err := newAuthInterceptor(ctx, toValidator(config.APIs.HTTP.Authorization.Config), logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create auth interceptor: %w", err)
	}
	listener.AddCloseFunc(closeAuth)

	server, err := NewHTTP(requestHandler, auth)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create http server interceptor: %w", err)
	}

	var wg sync.WaitGroup
	if !config.APIs.HTTP.PullDevices.Disabled {
		pdh := &pullDevicesHandler{
			s:                   store,
			isClient:            isClient,
			raClient:            raClient,
			devicesSubscription: devicesSubscription,
			subscriptionManager: subMgr,
			provider:            provider,
			triggerTask:         taskProcessor.Trigger,
		}
		runDevicePulling(ctx, pdh, config.APIs.HTTP.PullDevices.Interval, &wg)
	}
	runTaskProcessor(ctx, taskProcessor, subMgr, &wg)
	runSubscriptionManager(ctx, subMgr, config.Clients.Subscription.HTTP.ResubscribeInterval, &wg)

	return &Server{
		server:   server,
		listener: listener,
		doneWg:   &wg,
		cancel:   cancel,
	}, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Server) Serve() error {
	return s.server.Serve(s.listener)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	s.doneWg.Wait()
	return s.server.Shutdown(context.Background())
}

func runDevicePulling(ctx context.Context, pdh *pullDevicesHandler, timeout time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for pdh.runDevicePulling(ctx, timeout) {
		}
	}()
}

func runTaskProcessor(ctx context.Context, taskProcessor *TaskProcessor, subMgr *SubscriptionManager, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := taskProcessor.Run(ctx, subMgr); err != nil {
			if !kitNetGrpc.IsContextCanceled(err) {
				log.Errorf("failed to process subscriptionManager tasks: %w", err)
			}
		}
	}()
}

func runSubscriptionManager(ctx context.Context, subMgr *SubscriptionManager, interval time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		subMgr.Run(ctx, interval)
	}()
}
