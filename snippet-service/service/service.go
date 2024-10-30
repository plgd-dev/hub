package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	grpcService "github.com/plgd-dev/hub/v2/snippet-service/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/updater"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "snippet-service"

type Service struct {
	*service.Service

	store              store.Store
	resourceUpdater    *updater.ResourceUpdater
	resourceSubscriber *ResourceSubscriber
}

func createStore(ctx context.Context, config storeConfig.Config, onCheckForExpired func(), fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, error) {
	if config.Use != database.MongoDB {
		return nil, fmt.Errorf("invalid store use('%v')", config.Use)
	}
	s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("mongodb: %w", err)
	}
	if config.CleanUpExpiredUpdates != "" {
		scheduler, err := NewExpiredUpdatesChecker(config.CleanUpExpiredUpdates, config.ExtendCronParserBySeconds, onCheckForExpired)
		if err != nil {
			return nil, fmt.Errorf("cannot create scheduler: %w", err)
		}
		s.AddCloseFunc(func() {
			err2 := scheduler.Shutdown()
			if err2 != nil {
				log.Errorf("failed to shutdown scheduler: %w", err2)
			}
		})
	}
	return s, nil
}

func newHttpService(ctx context.Context, config HTTPConfig, validatorConfig validator.Config, tlsConfig certManagerServer.Config, ss *grpcService.SnippetServiceServer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*httpService.Service, func(), error) {
	httpValidator, err := validator.New(ctx, validatorConfig, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create http validator: %w", err)
	}
	httpService, err := httpService.New(serviceName, httpService.Config{
		Connection: listener.Config{
			Addr: config.Addr,
			TLS:  tlsConfig,
		},
		Authorization: validatorConfig,
		Server:        config.Server,
	}, ss, httpValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		httpValidator.Close()
		return nil, nil, fmt.Errorf("cannot create http service: %w", err)
	}
	return httpService, httpValidator.Close, nil
}

func newGrpcService(ctx context.Context, config grpcService.Config, ss *grpcService.SnippetServiceServer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*grpcService.Service, func(), error) {
	grpcValidator, err := validator.New(ctx, config.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	grpcService, err := grpcService.New(config, ss, grpcValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		grpcValidator.Close()
		return nil, nil, fmt.Errorf("cannot create grpc service: %w", err)
	}
	return grpcService, grpcValidator.Close, nil
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, serviceName, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	var closerFn fn.FuncList
	closerFn.AddFunc(otelClient.Close)
	tracerProvider := otelClient.GetTracerProvider()
	var resourceUpdater atomic.Pointer[updater.ResourceUpdater]

	db, err := createStore(ctx, config.Clients.Storage, func() {
		ru := resourceUpdater.Load()
		if ru == nil {
			return
		}
		ru.TimeoutPendingResourceUpdates()
	}, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	closerFn.AddFunc(func() {
		if errC := db.Close(ctx); errC != nil {
			log.Errorf("failed to close store: %w", errC)
		}
	})

	ru, err := updater.NewResourceUpdater(ctx, config.Clients.ResourceAggregate.Connection, db, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create resource change handler: %w", err)
	}
	resourceUpdater.Store(ru)

	closerFn.AddFunc(func() {
		errC := ru.Close()
		if errC != nil {
			log.Errorf("failed to close resource change handler: %w", errC)
		}
	})

	resourceSubscriber, err := NewResourceSubscriber(ctx, config.Clients.EventBus.NATS, config.Clients.EventBus.SubscriptionID, fileWatcher, logger, tracerProvider, resourceUpdater.Load())
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create resource subscriber: %w", err)
	}
	closerFn.AddFunc(func() {
		errC := resourceSubscriber.Close()
		if errC != nil {
			log.Errorf("failed to close resource subscriber: %w", errC)
		}
	})

	snippetService := grpcService.NewSnippetServiceServer(db, resourceUpdater.Load(), config.APIs.GRPC.Authorization.OwnerClaim, config.HubID, logger)

	grpcService, grpcServiceClose, err := newGrpcService(ctx, config.APIs.GRPC, snippetService, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, err
	}
	closerFn.AddFunc(grpcServiceClose)

	httpService, httpServiceClose, err := newHttpService(ctx, config.APIs.HTTP, config.APIs.GRPC.Authorization.Config, config.APIs.GRPC.TLS,
		snippetService, fileWatcher, logger, tracerProvider)
	if err != nil {
		grpcService.Close()
		closerFn.Execute()
		return nil, err
	}
	closerFn.AddFunc(httpServiceClose)

	s := service.New(grpcService, httpService)
	s.AddCloseFunc(closerFn.Execute)
	return &Service{
		Service: s,

		store:              db,
		resourceUpdater:    resourceUpdater.Load(),
		resourceSubscriber: resourceSubscriber,
	}, nil
}

func (s *Service) SnippetServiceStore() store.Store {
	return s.store
}

func (s *Service) CancelPendingResourceUpdates(ctx context.Context) error {
	return s.resourceUpdater.CancelPendingResourceUpdates(ctx)
}
