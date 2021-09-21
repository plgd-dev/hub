package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/fn"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/cloud/pkg/net/grpc/client"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/pkg/net/listener"
	cmClient "github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	natsClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
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

func newAuthInterceptor(ctx context.Context, config validator.Config, oauthCallbackPath string, logger log.Logger) (kitNetHttp.Interceptor, func(), error) {
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
			URI:    regexp.MustCompile(regexp.QuoteMeta(oauthCallbackPath)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(oauthCallbackPath)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.Events)),
		},
	}
	auth := kitNetHttp.NewInterceptorWithValidator(validator, authRules, whiteList...)

	return auth, fl.ToFunction(), nil
}

func newAuthorizationServiceClient(config AuthorizationServerConfig, logger log.Logger) (pbAS.AuthorizationServiceClient, func(), error) {
	asConn, err := grpcClient.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to authorization server: %w", err)
	}
	closeAsConn := func() {
		if err := asConn.Close(); err != nil && !kitNetGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during close connection to authorization server: %v", err)
		}
	}
	return pbAS.NewAuthorizationServiceClient(asConn.GRPC()), closeAsConn, nil
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

func newStore(ctx context.Context, config mongodb.Config, logger log.Logger) (*Store, func(), error) {
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

func newResourceDirectoryClient(config ResourceDirectoryConfig, logger log.Logger) (pbGRPC.GrpcGatewayClient, func(), error) {
	var fl fn.FuncList
	rdConn, err := grpcClient.New(config.Connection, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to resource directory: %w", err)
	}
	fl.AddFunc(func() {
		if err := rdConn.Close(); err != nil && !kitNetGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during closing of the connection to resource-directory: %w", err)
		}
	})
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn.GRPC())
	return rdClient, fl.ToFunction(), nil
}

func parseOAuthPaths(OAuthCallback string) (*url.URL, string, error) {
	oauthURL, err := url.Parse(OAuthCallback)
	if err != nil {
		return nil, "", fmt.Errorf("cannot parse oauth url: %w", err)
	}
	return oauthURL, OAuthCallback, nil
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

	rdClient, closeRdClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create to resource directory client: %w", err)
	}
	listener.AddCloseFunc(closeRdClient)

	sub, closeSub, err := newSubscriber(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create subscriber: %w", err)
	}
	listener.AddCloseFunc(closeSub)

	ctx, cancel := context.WithCancel(ctx)
	cleanUp.AddFunc(cancel)
	devicesSubscription := NewDevicesSubscription(ctx, rdClient, raClient, sub, config.Clients.Subscription.HTTP.ReconnectInterval)
	taskProcessor := NewTaskProcessor(raClient, config.TaskProcessor.MaxParallel, config.TaskProcessor.CacheSize,
		config.TaskProcessor.Timeout, config.TaskProcessor.Delay)

	asClient, closeAsClient, err := newAuthorizationServiceClient(config.Clients.AuthServer, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create authorization service client: %w", err)
	}
	listener.AddCloseFunc(closeAsClient)

	store, closeStore, err := newStore(ctx, config.Clients.Storage.MongoDB, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	listener.AddCloseFunc(closeStore)

	oauthURL, oauthCallback, err := parseOAuthPaths(config.APIs.HTTP.Authorization.RedirectURL)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot parse OAuth paths: %w", err)
	}

	subMgr := NewSubscriptionManager(config.APIs.HTTP.EventsURL, asClient, raClient, store, devicesSubscription,
		oauthCallback, taskProcessor.Trigger, config.Clients.Subscription.HTTP.ResubscribeInterval)

	provider, err := oauth2.NewPlgdProvider(ctx, config.APIs.HTTP.Authorization, logger, "sub")
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create device provider: %w", err)
	}
	listener.AddCloseFunc(provider.Close)

	requestHandler := NewRequestHandler(provider, subMgr, store, taskProcessor.Trigger)

	auth, closeAuth, err := newAuthInterceptor(ctx, toValidator(config.APIs.HTTP.Authorization), oauthURL.Path, logger)
	if err != nil {
		cleanUp.Execute()
		return nil, fmt.Errorf("cannot create auth interceptor: %w", err)
	}
	listener.AddCloseFunc(closeAuth)

	var wg sync.WaitGroup
	if !config.APIs.HTTP.PullDevices.Disabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for runDevicePulling(ctx, oauthCallback, config.APIs.HTTP.PullDevices.Interval, store, asClient, raClient, devicesSubscription, subMgr, taskProcessor.Trigger) {
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := taskProcessor.Run(ctx, subMgr); err != nil {
			if !kitNetGrpc.IsContextCanceled(err) {
				log.Errorf("failed to process subscriptionManager tasks: %w", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		subMgr.Run(ctx)
	}()

	server := Server{
		server:   NewHTTP(requestHandler, auth),
		listener: listener,
		doneWg:   &wg,
		cancel:   cancel,
	}

	return &server, nil
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

func runDevicePulling(ctx context.Context,
	oauthCallback string,
	timeout time.Duration,
	s *Store,
	asClient pbAS.AuthorizationServiceClient,
	raClient raService.ResourceAggregateClient,
	devicesSubscription *DevicesSubscription,
	subscriptionManager *SubscriptionManager,
	triggerTask OnTaskTrigger,
) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := pullDevices(ctx, s, asClient, raClient, devicesSubscription, subscriptionManager, oauthCallback, triggerTask)
	if err != nil {
		log.Errorf("cannot pull devices: %v", err)
	}
	<-ctx.Done()
	return ctx.Err() != context.Canceled
}
