package service

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	clientIS "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
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
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	eventstoreConfig "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/config"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.opentelemetry.io/otel/trace"
)

func createEvenstore(ctx context.Context, config eventstoreConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (eventstore.EventStore, error) {
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, tracerProvider, cqldb.WithUnmarshaler(utils.Unmarshal), cqldb.WithMarshaler(utils.Marshal))
		if err != nil {
			return nil, fmt.Errorf("cqldb: %w", err)
		}
		return s, nil
	}
	return nil, fmt.Errorf("invalid eventstore use('%v')", config.Use)
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "resource-aggregate", fileWatcher, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	otelClient.AddCloseFunc(cancel)
	tracerProvider := otelClient.GetTracerProvider()

	eventstore, err := createEvenstore(ctx, config.Clients.Eventstore.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create eventstore %w", err)
	}
	closeEventStore := func() {
		errC := eventstore.Close(ctx)
		if errC != nil {
			logger.Errorf("error occurs during closing of connection to eventstore: %w", errC)
		}
	}
	naClient, err := natsClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closeEventStore()
		otelClient.Close()
		return nil, fmt.Errorf("cannot create nats client %w", err)
	}
	opts := []publisher.Option{publisher.WithMarshaler(utils.Marshal), publisher.WithFlusherTimeout(config.Clients.Eventbus.NATS.FlusherTimeout)}
	if config.Clients.Eventbus.NATS.LeadResourceType.IsEnabled() {
		lrt := config.Clients.Eventbus.NATS.LeadResourceType
		opts = append(opts, publisher.WithLeadResourceType(lrt.GetCompiledRegexFilter(), lrt.Filter, lrt.UseUUID))
	}
	publisher, err := publisher.New(naClient.GetConn(), config.Clients.Eventbus.NATS.JetStream, opts...)
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
	authInterceptor := server.NewAuth(validator, server.WithWhiteListedMethods(ResourceAggregate_UpdateServiceMetadata_FullMethodName))
	opts, err := server.MakeDefaultOptions(authInterceptor, logger, tracerProvider)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}

	grpcServer, err := server.New(config.BaseConfig, fileWatcher, logger, tracerProvider, nil, opts...)
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
func NewService(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, eventStore eventstore.EventStore, publisher cqrsEventBus.Publisher) (*service.Service, error) {
	grpcServer, err := newGrpcServer(ctx, config.APIs.GRPC, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, err
	}

	closeGrpcServerOnError := func(err error) error {
		var errors *multierror.Error
		errors = multierror.Append(errors, err)
		if err2 := grpcServer.Close(); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot close server: %w", err2))
		}
		return errors.ErrorOrNil()
	}

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, closeGrpcServerOnError(fmt.Errorf("cannot create identity-store client: %w", err))
	}
	grpcServer.AddCloseFunc(closeIsClient)

	nats, err := natsClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, closeGrpcServerOnError(fmt.Errorf("cannot create nats client: %w", err))
	}
	grpcServer.AddCloseFunc(nats.Close)

	ownerCache := clientIS.NewOwnerCache(config.APIs.GRPC.Authorization.OwnerClaim, config.APIs.GRPC.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})
	grpcServer.AddCloseFunc(ownerCache.Close)

	serviceHeartbeat := NewServiceHeartbeat(config, eventStore, publisher, logger)
	grpcServer.AddCloseFunc(serviceHeartbeat.Close)

	requestHandler := NewRequestHandler(config, eventStore, publisher, func(getCtx context.Context, _ string, deviceIDs []string) ([]string, error) {
		getAllDevices := len(deviceIDs) == 0
		if !getAllDevices {
			return ownerCache.GetSelectedDevices(getCtx, deviceIDs)
		}
		return ownerCache.GetDevices(getCtx)
	}, serviceHeartbeat, logger)
	RegisterResourceAggregateServer(grpcServer.Server, requestHandler)

	// ResourceAggregate needs to stop gracefully to ensure that all commands are processed.
	grpcServer.SetGracefulStop(true)

	return service.New(grpcServer), nil
}
