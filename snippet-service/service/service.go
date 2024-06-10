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
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	grpcService "github.com/plgd-dev/hub/v2/snippet-service/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/store/cqldb"
	"github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "snippet-service"

type Service struct {
	*service.Service

	resourceSubscriber *ResourceSubscriber
}

type changeHandler struct{}

func (h *changeHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceChanged
		if ev.EventType() != s.EventType() {
			continue
		}
		if err := ev.Unmarshal(&s); err != nil {
			return err
		}
		// TODO: handle resource changed event
	}
}

func createStore(ctx context.Context, config storeConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, error) {
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("cqldb: %w", err)
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

	ca, err := grpcService.NewSnippetServiceServer(config.APIs.GRPC.Authorization.OwnerClaim, config.HubID, dbStorage, logger)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create grpc %s server: %w", serviceName, err)
	}
	closerFn.AddFunc(func() {
		errC := ca.Close(ctx)
		if errC != nil {
			log.Errorf("failed to close grpc %s server: %w", serviceName, errC)
		}
	})

	resourceSubscriber, err := NewResourceSubscriber(ctx, config, fileWatcher, logger, &changeHandler{})
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

	httpValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http validator: %w", err)
	}
	closerFn.AddFunc(httpValidator.Close)
	httpService, err := httpService.New(serviceName, httpService.Config{
		Connection: listener.Config{
			Addr: config.APIs.HTTP.Addr,
			TLS:  config.APIs.GRPC.TLS,
		},
		Authorization: config.APIs.GRPC.Authorization.Config,
		Server:        config.APIs.HTTP.Server,
	}, ca, httpValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}
	grpcValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	grpcService, err := grpcService.New(config.APIs.GRPC, ca, grpcValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc service: %w", err)
	}
	s := service.New(httpService, grpcService)
	s.AddCloseFunc(closerFn.Execute)
	return &Service{
		Service:            s,
		resourceSubscriber: resourceSubscriber,
	}, nil
}
