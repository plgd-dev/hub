package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
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
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "snippet-service"

type Service struct {
	*service.Service

	snippetService     *grpcService.SnippetServiceServer
	resourceUpdater    *store.ResourceUpdater
	resourceSubscriber *ResourceSubscriber
}

func createStore(ctx context.Context, config storeConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, error) {
	if config.Use == database.MongoDB {
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	}
	return nil, fmt.Errorf("invalid store use('%v')", config.Use)
}

func newStore(ctx context.Context, config StorageConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, func(), error) {
	var fl fn.FuncList
	db, err := createStore(ctx, config.Embedded, fileWatcher, logger, tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, err
	}
	fl.AddFunc(func() {
		if errC := db.Close(ctx); errC != nil {
			log.Errorf("failed to close mongodb store: %w", errC)
		}
	})
	if config.CleanUpRecords == "" {
		return db, fl.ToFunction(), nil
	}
	// TODO: do we need a cron job?
	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local)) //nolint:gosmopolitan
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	_, err = s.NewJob(gocron.CronJob(config.CleanUpRecords, config.ExtendCronParserBySeconds), gocron.NewTask(func() {
		/*
			_, errDel := db.DeleteNonDeviceExpiredRecords(ctx, time.Now())
			if errDel != nil && !errors.Is(errDel, store.ErrNotSupported) {
				log.Errorf("failed to delete expired signing records: %w", errDel)
			}
		*/
	}))
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	fl.AddFunc(func() {
		if errS := s.Shutdown(); errS != nil {
			log.Errorf("failed to shutdown cron job: %w", errS)
		}
	})
	s.Start()
	return db, fl.ToFunction(), nil
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

	dbStorage, closeStore, err := newStore(ctx, config.Clients.Storage, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	closerFn.AddFunc(closeStore)

	resourceUpdater, err := store.NewResourceUpdater(ctx, config.Clients.ResourceAggregate.Connection, config.Clients.ResourceAggregate.PendingCommandsCheckInterval, dbStorage, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create resource change handler: %w", err)
	}
	closerFn.AddFunc(func() {
		errC := resourceUpdater.Close()
		if errC != nil {
			log.Errorf("failed to close resource change handler: %w", errC)
		}
	})

	resourceSubscriber, err := NewResourceSubscriber(ctx, config.Clients.EventBus.NATS, fileWatcher, logger, resourceUpdater)
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

	snippetService := grpcService.NewSnippetServiceServer(dbStorage, resourceUpdater, config.APIs.GRPC.Authorization.OwnerClaim, config.HubID, logger)

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

		snippetService:     snippetService,
		resourceUpdater:    resourceUpdater,
		resourceSubscriber: resourceSubscriber,
	}, nil
}

func (s *Service) SnippetServiceServer() *grpcService.SnippetServiceServer {
	return s.snippetService
}
