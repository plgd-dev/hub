package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"go.opentelemetry.io/otel/trace"
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
		Audience: c.Audience,
		Endpoints: []validator.AuthorityConfig{
			{
				Authority: c.Authority,
				HTTP:      c.HTTP,
			},
		},
	}
}

const serviceName = "cloud2cloud-connector"

func newAuthInterceptor(validator *validator.Validator) pkgHttpJwt.Interceptor {
	authRules := pkgHttp.NewDefaultAuthorizationRules(uri.API)

	whiteList := []pkgHttpJwt.RequestMatcher{
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
	auth := pkgHttpJwt.NewInterceptorWithValidator(validator, authRules, whiteList...)

	return auth
}

func newIdentityStoreClient(config IdentityStoreConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newSubscriber(config natsClient.ConfigSubscriber, fileWatcher *fsnotify.Watcher, logger log.Logger, tp trace.TracerProvider) (*subscriber.Subscriber, func(), error) {
	var fl fn.FuncList
	nats, err := natsClient.New(config.Config, fileWatcher, logger, tp)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	fl.AddFunc(nats.Close)

	sub, err := subscriber.New(nats.GetConn(),
		config.PendingLimits, config.LeadResourceType.IsEnabled(),
		logger, subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create subscriber: %w", err)
	}
	fl.AddFunc(sub.Close)

	return sub, fl.ToFunction(), nil
}

func newStore(ctx context.Context, config pkgMongo.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, func(), error) {
	var fl fn.FuncList
	certManager, err := cmClient.New(config.TLS, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	fl.AddFunc(certManager.Close)

	db, err := mongodb.NewStore(ctx, config, certManager.GetTLSConfig(), tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create mongodb subscription store: %w", err)
	}
	fl.AddFunc(func() {
		if errC := db.Close(ctx); errC != nil {
			log.Errorf("failed to close subscription store: %w", errC)
		}
	})

	store, err := NewStore(ctx, db)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cloud-to-cloud store: %w", err)
	}

	return store, fl.ToFunction(), nil
}

func newGrpcGatewayClient(config GrpcGatewayConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbGRPC.GrpcGatewayClient, func(), error) {
	var fl fn.FuncList
	grpcConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newResourceAggregateClient(config ResourceAggregateConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (raService.ResourceAggregateClient, func(), error) {
	var fl fn.FuncList
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
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

func newDevicesSubscription(ctx context.Context, config Config, raClient raService.ResourceAggregateClient, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*DevicesSubscription, func(), error) {
	var fl fn.FuncList

	grpcClient, closeGrpcClient, err := newGrpcGatewayClient(config.Clients.GrpcGateway, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create to grpc-gateway client: %w", err)
	}
	fl.AddFunc(closeGrpcClient)

	sub, closeSub, err := newSubscriber(config.Clients.Eventbus.NATS, fileWatcher, logger, tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create subscriber: %w", err)
	}
	fl.AddFunc(closeSub)

	devicesSubscription := NewDevicesSubscription(ctx, tracerProvider, grpcClient, raClient, sub, config.Clients.Subscription.HTTP.ReconnectInterval)
	return devicesSubscription, fl.ToFunction(), nil
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector.Config, serviceName, fileWatcher, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	otelClient.AddCloseFunc(cancel)

	tracerProvider := otelClient.GetTracerProvider()

	listener, err := listener.New(config.APIs.HTTP.Connection, fileWatcher, logger)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}
	listener.AddCloseFunc(otelClient.Close)
	var cleanUp fn.FuncList
	cleanUp.AddFunc(func() {
		if errC := listener.Close(); errC != nil {
			logger.Errorf("cannot create http server: %w", errC)
		}
	})

	raClient, closeRaClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, fileWatcher, logger, tracerProvider)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create to resource aggregate client: %w", err)
	}
	listener.AddCloseFunc(closeRaClient)
	devicesSubscription, closeDevSub, err := newDevicesSubscription(ctx, config, raClient, fileWatcher, logger, tracerProvider)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create devices subscription subscriber: %w", err)
	}
	listener.AddCloseFunc(closeDevSub)

	taskProcessor := NewTaskProcessor(raClient, tracerProvider, config.TaskProcessor.MaxParallel, config.TaskProcessor.CacheSize,
		config.TaskProcessor.Timeout, config.TaskProcessor.Delay)

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	listener.AddCloseFunc(closeIsClient)

	store, closeStore, err := newStore(ctx, config.Clients.Storage.MongoDB, fileWatcher, logger, tracerProvider)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	listener.AddCloseFunc(closeStore)

	validator, err := validator.New(ctx, toValidator(config.APIs.HTTP.Authorization.Config), fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	listener.AddCloseFunc(validator.Close)

	provider, err := oauth2.NewPlgdProvider(ctx, config.APIs.HTTP.Authorization.Config, fileWatcher, logger, tracerProvider, "", "", validator.GetParser())
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create device provider: %w", err)
	}
	listener.AddCloseFunc(provider.Close)

	subMgr := NewSubscriptionManager(config.APIs.HTTP.EventsURL, isClient, raClient, store, devicesSubscription,
		provider, taskProcessor.Trigger, tracerProvider)

	requestHandler := NewRequestHandler(config.APIs.HTTP.Authorization.OwnerClaim, provider, subMgr, store, taskProcessor.Trigger, tracerProvider)

	auth := newAuthInterceptor(validator)

	httpHandler, err := NewHTTP(requestHandler, auth, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create http server interceptor: %w", err)
	}

	httpServer := http.Server{
		Handler:           pkgHttp.OpenTelemetryNewHandler(httpHandler, serviceName, tracerProvider),
		ReadTimeout:       config.APIs.HTTP.Server.ReadTimeout,
		ReadHeaderTimeout: config.APIs.HTTP.Server.ReadHeaderTimeout,
		WriteTimeout:      config.APIs.HTTP.Server.WriteTimeout,
		IdleTimeout:       config.APIs.HTTP.Server.IdleTimeout,
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
			tracerProvider:      tracerProvider,
		}
		runDevicePulling(ctx, pdh, config.APIs.HTTP.PullDevices.Interval, &wg)
	}
	runTaskProcessor(ctx, taskProcessor, subMgr, &wg)
	runSubscriptionManager(ctx, subMgr, config.Clients.Subscription.HTTP.ResubscribeInterval, &wg)

	return &Server{
		server:   &httpServer,
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
func (s *Server) Close() error {
	s.cancel()
	s.doneWg.Wait()
	return s.server.Shutdown(context.Background())
}

func runDevicePulling(ctx context.Context, pdh *pullDevicesHandler, timeout time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if !pdh.runDevicePulling(ctx, timeout) {
				break
			}
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
