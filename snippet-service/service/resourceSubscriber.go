package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.opentelemetry.io/otel/trace"
)

type resourceChangedHandler struct {
	storage  store.Store
	raConn   *grpcClient.Client
	raClient raService.ResourceAggregateClient
	logger   log.Logger
}

func newResourceChangedHandler(config ResourceAggregateConfig, storage store.Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*resourceChangedHandler, error) {
	var fl fn.FuncList
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	fl.AddFunc(func() {
		if err := raConn.Close(); err != nil && !pkgGrpc.IsContextCanceled(err) {
			logger.Errorf("error occurs during closing of the connection to resource-aggregate: %w", err)
		}
	})
	return &resourceChangedHandler{
		storage:  storage,
		raConn:   raConn,
		raClient: raService.NewResourceAggregateClient(raConn.GRPC()),
		logger:   logger,
	}, nil
}

func (h *resourceChangedHandler) getConditions(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string) ([]*pb.Condition, error) {
	conditions := make([]*pb.Condition, 0, 4)
	err := h.storage.GetLatestConditions(ctx, owner, &store.GetLatestConditionsQuery{
		DeviceID:           deviceID,
		ResourceHref:       resourceHref,
		ResourceTypeFilter: resourceTypes,
	}, func(v *store.Condition) error {
		c, errG := v.GetLatest()
		if errG != nil {
			return fmt.Errorf("cannot get condition: %w", errG)
		}
		conditions = append(conditions, c.Clone())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get latest conditions: %w", err)
	}

	// TODO: evaluate conditions
	// https://github.com/itchyny/gojq

	return conditions, nil
}

type configurationWithTokens struct {
	configuration *pb.Configuration
	tokens        []string
}

func (h *resourceChangedHandler) getConfigurationsWithTokens(ctx context.Context, owner string, conditions []*pb.Condition) (map[string]configurationWithTokens, error) {
	confTokens := make(map[string][]string)
	idFilter := make([]*pb.IDFilter, 0, len(conditions))
	for _, c := range conditions {
		confID := c.GetConfigurationId()
		if confID == "" {
			h.logger.Warnf("invalid condition(%v)", c)
			continue
		}
		tokens := confTokens[confID]
		tokens = append(tokens, c.GetApiAccessToken())
		confTokens[confID] = tokens
		idFilter = append(idFilter, &pb.IDFilter{
			Id: confID,
			Version: &pb.IDFilter_Latest{
				Latest: true,
			},
		})
	}
	if (len(idFilter)) == 0 {
		return map[string]configurationWithTokens{}, nil
	}

	// get configurations
	configurations := make([]*pb.Configuration, 0, 4)
	err := h.storage.GetConfigurations(ctx, owner, &pb.GetConfigurationsRequest{
		IdFilter: idFilter,
	}, func(v *store.Configuration) error {
		c, errG := v.GetLatest()
		if errG != nil {
			return fmt.Errorf("cannot get configuration: %w", errG)
		}
		configurations = append(configurations, c.Clone())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get configurations: %w", err)
	}

	confsWithTokens := make(map[string]configurationWithTokens)
	for _, c := range configurations {
		tokens := strings.Unique(confTokens[c.GetId()])
		if len(tokens) == 0 {
			h.logger.Errorf("no tokens found for configuration(id:%v)", c.GetId())
			continue
		}
		confsWithTokens[c.GetId()] = configurationWithTokens{
			configuration: c,
			tokens:        tokens,
		}
	}

	return confsWithTokens, nil
}

func (h *resourceChangedHandler) applyConfigurations(ctx context.Context, rc *events.ResourceChanged) error {
	owner := rc.GetAuditContext().GetOwner()

	if owner == "" {
		return errors.New("owner not set")
	}

	resourceID := rc.GetResourceId()
	deviceID := resourceID.GetDeviceId()
	resourceHref := resourceID.GetHref()
	resourceTypes := rc.GetResourceTypes()
	// get matching conditions
	conditions, err := h.getConditions(ctx, owner, deviceID, resourceHref, resourceTypes)
	if err != nil {
		return err
	}

	// get configurations with tokens
	confsWithTokens, err := h.getConfigurationsWithTokens(ctx, owner, conditions)
	if err != nil {
		return err
	}

	// apply configurations
	var errors *multierror.Error
	for confID, c := range confsWithTokens {
		for _, cr := range c.configuration.GetResources() {
			if cr.GetHref() != resourceHref {
				continue
			}

			// insert applied configuration
			/// - ak force tak upsert

			// CorrelationId = vygenerovat uuid.NewString, cez InvokeConfiguration sa moze nastavit
			//

			for _, token := range c.tokens {
				h.logger.Infof("applying configuration(id:%v) to resource(%v)", confID, resourceHref)
				ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)
				upd := &commands.UpdateResourceRequest{
					ResourceId:    resourceID,
					CorrelationId: "snippet-service configuration apply",
					Content:       cr.GetContent(),
					TimeToLive:    cr.GetTimeToLive(),
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: confID,
					},
				}
				_, err := h.raClient.UpdateResource(ctxWithToken, upd)
				if err != nil {
					errors = multierror.Append(errors, err)
					continue
				}

				// v response je validUntil, ak uplynie cas a zostane v stave pending tak nastavit timeout

				// zapis do AppliedConfigurations

				h.logger.Infof("configuration(id:%v) applied to resource(%v)", confID, resourceHref)
				break
			}
			// TODO: write applied configuration to storage
			// ak sa nepodari ziadny update tak zapisat ResourceUpdated so statusom chybou
		}
	}
	return errors.ErrorOrNil()
}

func (h *resourceChangedHandler) Handle(ctx context.Context, iter eventbus.Iter) error {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceChanged
		if ev.EventType() != s.EventType() {
			h.logger.Errorf("unexpected event type: %v", ev.EventType())
			continue
		}
		if err := ev.Unmarshal(&s); err != nil {
			h.logger.Errorf("cannot unmarshal event: %w", err)
			continue
		}
		h.logger.Infof("resource change received: %v", &s)
		if err := h.applyConfigurations(ctx, &s); err != nil {
			h.logger.Errorf("cannot apply configurations: %w", err)
		}
	}
}

func (h *resourceChangedHandler) Close() error {
	return h.raConn.Close()
}

type ResourceSubscriber struct {
	natsClient          *natsClient.Client
	subscriptionHandler eventbus.Handler
	subscriber          *subscriber.Subscriber
	observer            eventbus.Observer
}

func WithAllDevicesAndResources() func(values map[string]string) {
	return func(values map[string]string) {
		values[utils.DeviceIDKey] = "*"
		values[utils.HrefIDKey] = "*"
	}
}

func NewResourceSubscriber(ctx context.Context, config natsClient.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, handler eventbus.Handler) (*ResourceSubscriber, error) {
	nats, err := natsClient.New(config, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}

	subscriber, err := subscriber.New(nats.GetConn(),
		config.PendingLimits,
		logger,
		subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource subscriber: %w", err)
	}

	subscriptionID := uuid.NewString()
	const owner = "*"
	subject := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent,
		isEvents.WithOwner(owner),
		WithAllDevicesAndResources(),
		isEvents.WithEventType((&events.ResourceChanged{}).EventType()))}
	observer, err := subscriber.Subscribe(ctx, subscriptionID, subject, handler)
	if err != nil {
		subscriber.Close()
		nats.Close()
		return nil, fmt.Errorf("cannot subscribe to resource change events: %w", err)
	}

	return &ResourceSubscriber{
		natsClient:          nats,
		subscriptionHandler: handler,
		subscriber:          subscriber,
		observer:            observer,
	}, nil
}

func (r *ResourceSubscriber) Close() error {
	err := r.observer.Close()
	r.subscriber.Close()
	r.natsClient.Close()
	return err
}
