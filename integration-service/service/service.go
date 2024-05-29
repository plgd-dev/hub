package service

import (
	"context"
	"fmt"

	//"time"

	//"github.com/go-co-op/gocron/v2"
	grpcService "github.com/plgd-dev/hub/v2/integration-service/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/integration-service/service/http"
	"github.com/plgd-dev/hub/v2/integration-service/store"
	storeConfig "github.com/plgd-dev/hub/v2/integration-service/store/config"
	"github.com/plgd-dev/hub/v2/integration-service/store/cqldb"
	"github.com/plgd-dev/hub/v2/integration-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "integration-service"

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
		if err := db.Close(ctx); err != nil {
			log.Errorf("failed to close mongodb store: %w", err)
		}
	})
	if config.CleanUpRecords == "" {
		return db, fl.ToFunction(), nil
	}

	// s := gocron.NewScheduler(time.Local)
	// if config.ExtendCronParserBySeconds {
	// 	s = s.CronWithSeconds(config.CleanUpRecords)
	// } else {
	// 	s = s.Cron(config.CleanUpRecords)
	// }
	// _, err = s.Do(func() {
	// 	/*
	// 		_, errDel := db.DeleteNonDeviceExpiredRecords(ctx, time.Now())
	// 		if errDel != nil && !errors.Is(errDel, store.ErrNotSupported) {
	// 			log.Errorf("failed to delete expired signing records: %w", errDel)
	// 		}
	// 	*/
	// })
	// if err != nil {
	// 	fl.Execute()
	// 	return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	// }
	// fl.AddFunc(s.Clear)
	// fl.AddFunc(s.Stop)
	// s.StartAsync()

	return db, fl.ToFunction(), nil
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "certificate-authority", fileWatcher, logger)
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

	ca, err := grpcService.NewIntegrationServiceServer(config.APIs.GRPC.Authorization.OwnerClaim, config.HubID, dbStorage, logger)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create grpc certificate authority server: %w", err)
	}
	closerFn.AddFunc(ca.Close)

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
		Server: config.APIs.HTTP.Server,
	}, ca, httpValidator, fileWatcher, logger, tracerProvider)

	if err != nil {
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
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}

	s := service.New(httpService, grpcService)
	s.AddCloseFunc(closerFn.Execute)

	return s, nil
}

type RequestHandler struct {
	msg string
}
