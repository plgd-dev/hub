package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
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
	"google.golang.org/grpc/codes"
)

type resourceChangedHandler struct {
	storage  store.Store
	raConn   *grpcClient.Client
	raClient raService.ResourceAggregateClient
	logger   log.Logger
}

func newResourceChangedHandler(config ResourceAggregateConfig, storage store.Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*resourceChangedHandler, error) {
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	return &resourceChangedHandler{
		storage:  storage,
		raConn:   raConn,
		raClient: raService.NewResourceAggregateClient(raConn.GRPC()),
		logger:   logger,
	}, nil
}

func (h *resourceChangedHandler) getConditions(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string) ([]*pb.Condition, error) {
	conditions := make([]*pb.Condition, 0, 4)
	err := h.storage.GetLatestEnabledConditions(ctx, owner, &store.GetLatestConditionsQuery{
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

type configurationWithConditions struct {
	configuration *pb.Configuration
	conditions    []*pb.Condition
}

func (h *resourceChangedHandler) getConfigurations(ctx context.Context, owner string, conditions []*pb.Condition) ([]configurationWithConditions, error) {
	confsToConditions := make(map[string][]*pb.Condition)
	idFilter := make([]*pb.IDFilter, 0, len(conditions))
	for _, c := range conditions {
		confID := c.GetConfigurationId()
		if confID == "" {
			h.logger.Warnf("invalid condition(%v)", c)
			continue
		}
		if c.GetApiAccessToken() == "" {
			h.logger.Warnf("skipping condition(%v) with no token", c)
			continue
		}
		confConditions := confsToConditions[confID]
		confConditions = append(confConditions, c)
		confsToConditions[confID] = confConditions
		idFilter = append(idFilter, &pb.IDFilter{
			Id: confID,
			Version: &pb.IDFilter_Latest{
				Latest: true,
			},
		})
	}
	if (len(idFilter)) == 0 {
		return []configurationWithConditions{}, nil
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

	confsWithConditions := make([]configurationWithConditions, 0, len(configurations))
	for _, c := range configurations {
		confConditions := confsToConditions[c.GetId()]
		if len(confConditions) == 0 {
			h.logger.Errorf("no conditions found for configuration(id:%v)", c.GetId())
			continue
		}
		slices.SortFunc(confConditions, func(i, j *pb.Condition) int {
			return cmp.Compare(i.GetApiAccessToken(), j.GetApiAccessToken())
		})
		confConditions = slices.CompactFunc(confConditions, func(i, j *pb.Condition) bool {
			return i.GetApiAccessToken() == j.GetApiAccessToken()
		})
		confsWithConditions = append(confsWithConditions, configurationWithConditions{
			configuration: c,
			conditions:    confConditions,
		})
	}

	return confsWithConditions, nil
}

func (h *resourceChangedHandler) applyConfigurationToResource(ctx context.Context, deviceID, configurationID string, cr *pb.Configuration_Resource, conditions []*pb.Condition) error {
	// insert applied configuration
	/// - ak force tak upsert
	var errs *multierror.Error

	href := cr.GetHref()
	resourceID := &commands.ResourceId{Href: href, DeviceId: deviceID}
	for i, cond := range conditions {
		h.logger.Debugf("applying configuration(id:%v) to resource(%v) with token(%v)", configurationID, href, i)
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, cond.GetApiAccessToken())
		upd := &commands.UpdateResourceRequest{
			ResourceId:    resourceID,
			CorrelationId: uuid.NewString(),
			Content:       cr.GetContent(),
			TimeToLive:    cr.GetTimeToLive(),
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: configurationID,
			},
		}
		_, err := h.raClient.UpdateResource(ctxWithToken, upd)
		if err != nil {
			if grpcCode := pkgGrpc.ErrToStatus(err).Code(); grpcCode == codes.Unauthenticated {
				continue
			}
			errs = multierror.Append(errs, err)
			h.logger.Errorf("cannot apply configuration(id:%v) to resource(%v): %w", configurationID, href, err)
			// zapis error AppliedConfigurations
			break
		}

		h.logger.Infof("configuration(id:%v) applied to resource(%v)", configurationID, href)

		// v response je validUntil, ak uplynie cas a zostane v stave pending tak nastavit timeout

		// zapis do AppliedConfigurations

		return nil
	}
	// TODO: write applied configuration to storage
	// ak sa nepodari ziadny update tak zapisat ResourceUpdated so statusom chybou

	return errs.ErrorOrNil()
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
	confsWithConditions, err := h.getConfigurations(ctx, owner, conditions)
	if err != nil {
		return err
	}
	if len(confsWithConditions) == 0 {
		return nil
	}

	// apply configurations
	// TODO: what to do with multiple configurations?
	// currently we apply just the first
	var confWithConditions *configurationWithConditions
	for i, c := range confsWithConditions {
		if len(c.configuration.GetResources()) > 0 {
			confWithConditions = &confsWithConditions[i]
			break
		}
	}

	if confWithConditions == nil {
		return nil
	}

	resources := []*pb.AppliedDeviceConfiguration_Resource{}
	for _, cr := range confWithConditions.configuration.GetResources() {
		resources = append(resources, &pb.AppliedDeviceConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: uuid.NewString(),
			Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
		})
	}

	_, err = h.storage.CreateAppliedDeviceConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Owner:    owner,
		DeviceId: deviceID,
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id:      confWithConditions.configuration.GetId(),
			Version: confWithConditions.configuration.GetVersion(),
		},
		ExecutedBy: &pb.AppliedDeviceConfiguration_ConditionId{
			ConditionId: &pb.AppliedDeviceConfiguration_RelationTo{
				Id: confWithConditions.conditions[0].GetId(),
			},
		},
		Resources: resources,
		Timestamp: time.Now().UnixNano(),
	})
	if err != nil {
		if store.IsDuplicateKeyError(err) {
			// applied configuration already exists
			h.logger.Debugf("applied configuration already exists for resource(%s)", resourceHref)
			return nil
		}
		return fmt.Errorf("cannot create applied device configuration: %w", err)
	}

	var errs *multierror.Error
	for _, cr := range confWithConditions.configuration.GetResources() {
		err := h.applyConfigurationToResource(ctx, deviceID, confWithConditions.configuration.GetId(), cr, confWithConditions.conditions)
		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()
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
