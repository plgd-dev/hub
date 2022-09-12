package service

import (
	"context"
	"fmt"

	clientIS "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
	cqrsEventBus "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	cqrsEventStore "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	cqrsMaintenance "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/maintenance"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.opentelemetry.io/otel/trace"
)

type EventStore interface {
	cqrsEventStore.EventStore
	cqrsMaintenance.EventStore
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "resource-aggregate", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	tracerProvider := otelClient.GetTracerProvider()

	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, tracerProvider, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	closeEventStore := func() {
		err := eventstore.Close(ctx)
		if err != nil {
			logger.Errorf("error occurs during closing of connection to mongodb: %w", err)
		}
	}
	naClient, err := natsClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger)
	if err != nil {
		closeEventStore()
		otelClient.Close()
		return nil, fmt.Errorf("cannot create nats client %w", err)
	}
	publisher, err := publisher.New(naClient.GetConn(), config.Clients.Eventbus.NATS.JetStream, publisher.WithMarshaler(utils.Marshal), publisher.WithFlusherTimeout(config.Clients.Eventbus.NATS.Config.FlusherTimeout))
	if err != nil {
		naClient.Close()
		closeEventStore()
		otelClient.Close()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	naClient.AddCloseFunc(otelClient.Close)
	naClient.AddCloseFunc(publisher.Close)

	service, err := NewService(ctx, config, fileWatcher, logger, tracerProvider, eventstore, publisher)
	if err != nil {
		naClient.Close()
		closeEventStore()
		return nil, fmt.Errorf("cannot create nats publisher %w", err)
	}
	service.AddCloseFunc(closeEventStore)
	service.AddCloseFunc(naClient.Close)

	return service, nil
}

func newGrpcServer(ctx context.Context, config GRPCConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*server.Server, error) {
	validator, err := validator.New(ctx, config.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	authInterceptor := server.NewAuth(validator)
	opts, err := server.MakeDefaultOptions(authInterceptor, logger, tracerProvider)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	grpcServer, err := server.New(config.Config, fileWatcher, logger, opts...)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	grpcServer.AddCloseFunc(validator.Close)
	return grpcServer, nil
}

func newIdentityStoreClient(config IdentityStoreConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := client.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to identity-store: %w", err)
	}
	closeIsConn := func() {
		if err := isConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to identity-store: %w", err)
		}
	}
	isClient := pbIS.NewIdentityStoreClient(isConn.GRPC())
	return isClient, closeIsConn, nil
}

// New creates new Server with provided store and publisher.
func NewService(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, eventStore EventStore, publisher cqrsEventBus.Publisher) (*service.Service, error) {
	grpcServer, err := newGrpcServer(ctx, config.APIs.GRPC, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, err
	}

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		err = fmt.Errorf("cannot create identity-store client: %w", err)
		err2 := grpcServer.Close()
		if err2 != nil {
			err = fmt.Errorf(`[%w, "cannot close server: %v"]`, err, err2)
		}
		return nil, err
	}
	grpcServer.AddCloseFunc(closeIsClient)

	nats, err := natsClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger)
	if err != nil {
		err = fmt.Errorf("cannot create nats client: %w", err)
		err2 := grpcServer.Close()
		if err2 != nil {
			err = fmt.Errorf(`[%w, "cannot close server: %v"]`, err, err2)
		}
		return nil, err
	}
	grpcServer.AddCloseFunc(nats.Close)

	ownerCache := clientIS.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})
	grpcServer.AddCloseFunc(ownerCache.Close)

	requestHandler := NewRequestHandler(config, eventStore, publisher, func(ctx context.Context, owner string, deviceIDs []string) ([]string, error) {
		getAllDevices := len(deviceIDs) == 0
		if !getAllDevices {
			return ownerCache.GetSelectedDevices(ctx, deviceIDs)
		}
		return ownerCache.GetDevices(ctx)
	})
	RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

	return service.New(grpcServer), nil
}
