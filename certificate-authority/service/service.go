package service

import (
	"context"
	"fmt"
	"time"

	gocron "github.com/go-co-op/gocron"
	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/certificate-authority/service/http"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "certificate-authority"

func newStore(ctx context.Context, config StorageConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*mongodb.Store, func(), error) {
	var fl fn.FuncList

	certManager, err := cmClient.New(config.MongoDB.Mongo.TLS, fileWatcher, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	fl.AddFunc(certManager.Close)

	db, err := mongodb.NewStore(ctx, config.MongoDB, certManager.GetTLSConfig(), logger, tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create mongodb store: %w", err)
	}
	fl.AddFunc(func() {
		if err := db.Close(ctx); err != nil {
			log.Errorf("failed to close mongodb store: %w", err)
		}
	})
	if config.CleanUpRecords == "" {
		return db, fl.ToFunction(), nil
	}
	s := gocron.NewScheduler(time.Local)
	if config.ExtendCronParserAboutSeconds {
		s = s.CronWithSeconds(config.CleanUpRecords)
	} else {
		s = s.Cron(config.CleanUpRecords)
	}
	_, err = s.Do(func() {
		_, errDel := db.DeleteNonDeviceExpiredRecords(ctx, time.Now())
		if errDel != nil {
			log.Errorf("failed to delete expired signing records: %w", errDel)
		}
	})
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	fl.AddFunc(s.Stop)
	fl.AddFunc(s.Clear)
	s.StartAsync()

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

	ca, err := grpcService.NewCertificateAuthorityServer(config.APIs.GRPC.Authorization.OwnerClaim, config.Signer, dbStorage, logger)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
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
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	s := service.New(httpService, grpcService)
	s.AddCloseFunc(closerFn.Execute)
	return s, nil
}
