package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/itchyny/gojq"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
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
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
)

type pendingConfiguration struct {
	id         string
	owner      string
	resourceID *commands.ResourceId
}

type resourceChangedHandler struct {
	storage               store.Store
	raConn                *grpcClient.Client
	raClient              raService.ResourceAggregateClient
	pendingConfigurations *cache.Cache[uuid.UUID, *pendingConfiguration]
	logger                log.Logger
}

func newPendingConfigurationsCache(ctx context.Context, interval time.Duration) *cache.Cache[uuid.UUID, *pendingConfiguration] {
	c := cache.NewCache[uuid.UUID, *pendingConfiguration]()
	add := periodic.New(ctx.Done(), interval)
	add(func(now time.Time) bool {
		c.CheckExpirations(now)
		return true
	})
	return c
}

func newResourceChangedHandler(ctx context.Context, config ResourceAggregateConfig, storage store.Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*resourceChangedHandler, error) {
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	return &resourceChangedHandler{
		storage:               storage,
		raConn:                raConn,
		raClient:              raService.NewResourceAggregateClient(raConn.GRPC()),
		pendingConfigurations: newPendingConfigurationsCache(ctx, config.PendingCommandsCheckInterval),
		logger:                logger,
	}, nil
}

type evaluateCondition = func(condition *pb.Condition) bool

func (h *resourceChangedHandler) getConditions(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string, eval evaluateCondition) ([]*pb.Condition, error) {
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
		if !eval(c) {
			h.logger.Debugf("condition(%v) skipped", c)
			return nil
		}
		conditions = append(conditions, c.Clone())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get latest conditions: %w", err)
	}
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

type appliedCondition struct {
	id      string
	version uint64
	token   string
}

func (h *resourceChangedHandler) applyConfigurationToResource(ctx context.Context, resourceID *commands.ResourceId, configurationID string, cr *pb.Configuration_Resource, conditionID, token string) (int64, error) {
	h.logger.Debugf("applying configuration(id:%v) to resource(%v) with condition(%v)", configurationID, resourceID.GetHref(), conditionID)
	upd := &commands.UpdateResourceRequest{
		ResourceId:    resourceID,
		CorrelationId: uuid.NewString(),
		Content:       cr.GetContent(),
		TimeToLive:    cr.GetTimeToLive(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: configurationID,
		},
	}
	ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)
	res, err := h.raClient.UpdateResource(ctxWithToken, upd)
	if err != nil {
		return 0, err
	}
	h.logger.Infof("configuration(id:%v) applied to resource(%v)", configurationID, resourceID.GetHref())

	return res.GetValidUntil(), nil
}

func (h *resourceChangedHandler) findTokenAndApplyConfigurationToResource(ctx context.Context, resourceID *commands.ResourceId, configurationID string, cr *pb.Configuration_Resource, conditions []*pb.Condition) (appliedCondition, int64, error) {
	for _, cond := range conditions {
		condID := cond.GetId()
		token := cond.GetApiAccessToken()
		validUntil, err := h.applyConfigurationToResource(ctx, resourceID, configurationID, cr, condID, token)
		if err != nil {
			if grpcCode := pkgGrpc.ErrToStatus(err).Code(); grpcCode == codes.Unauthenticated {
				h.logger.Debugf("cannot apply configuration(id:%v) to resource(%v): invalid token", configurationID, resourceID.GetHref())
				continue
			}
			h.logger.Errorf("cannot apply configuration(id:%v) to resource(%v): %w", configurationID, resourceID.GetHref(), err)
			return appliedCondition{}, 0, err
		}
		return appliedCondition{id: condID, version: cond.GetVersion(), token: token}, validUntil, nil
	}
	return appliedCondition{}, 0, errors.New("cannot apply configuration: no valid token found")
}

func (h *resourceChangedHandler) updateAppliedConfigurationPendingResources(ctx context.Context, id, owner, href string, status pb.AppliedDeviceConfiguration_Resource_Status) {
	err := h.storage.UpdateAppliedConfigurationPendingResources(ctx, &store.UpdateAppliedConfigurationPendingResourceRequest{
		ID:     id,
		Owner:  owner,
		Href:   href,
		Status: status,
	})
	if err != nil {
		h.logger.Errorf("cannot update applied configuration pending resource(%v): %w", href, err)
	}
}

func (h *resourceChangedHandler) applyConfigurationToResources(ctx context.Context, owner, deviceID string, confWithConditions *configurationWithConditions) error {
	resources := map[string]*pb.AppliedDeviceConfiguration_Resource{}
	for _, cr := range confWithConditions.configuration.GetResources() {
		resources[cr.GetHref()] = &pb.AppliedDeviceConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: uuid.NewString(),
			Status:        pb.AppliedDeviceConfiguration_Resource_QUEUED,
		}
	}

	/// TODO: ak force tak upsert
	appliedConf, errC := h.storage.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Owner:    owner,
		DeviceId: deviceID,
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id:      confWithConditions.configuration.GetId(),
			Version: confWithConditions.configuration.GetVersion(),
		},
		ExecutedBy: &pb.AppliedDeviceConfiguration_ConditionId{
			ConditionId: &pb.AppliedDeviceConfiguration_RelationTo{
				Id:      confWithConditions.conditions[0].GetId(),
				Version: confWithConditions.conditions[0].GetVersion(),
			},
		},
		Resources: maps.Values(resources),
		Timestamp: time.Now().UnixNano(),
	})
	if errC != nil {
		if store.IsDuplicateKeyError(errC) {
			// applied configuration already exists
			h.logger.Debugf("applied configuration already exists for device(%s) and configuration(%s)", deviceID,
				confWithConditions.configuration.GetId())
			return nil
		}
		return fmt.Errorf("cannot create applied device configuration: %w", errC)
	}

	var appliedCond appliedCondition
	for _, cr := range confWithConditions.configuration.GetResources() {
		href := cr.GetHref()
		resourceID := &commands.ResourceId{Href: href, DeviceId: deviceID}
		confID := confWithConditions.configuration.GetId()
		var validUntil int64
		var errA error
		if appliedCond.id != "" {
			validUntil, errA = h.applyConfigurationToResource(ctx, resourceID, confID, cr, appliedCond.id, appliedCond.token)
		} else {
			appliedCond, validUntil, errA = h.findTokenAndApplyConfigurationToResource(ctx, resourceID, confID, cr, confWithConditions.conditions)
			if errA == nil {
				appliedConf.ExecutedBy = pb.MakeExecutedByConditionId(appliedCond.id, appliedCond.version)
			}
		}

		resource := resources[href]
		if errA == nil {
			resource.Status = pb.AppliedDeviceConfiguration_Resource_PENDING
			h.pendingConfigurations.LoadOrStore(
				resourceID.ToUUID(), cache.NewElement(
					&pendingConfiguration{
						id:         appliedConf.GetId(),
						owner:      owner,
						resourceID: resourceID,
					},
					pkgTime.Unix(0, validUntil),
					func(d *pendingConfiguration) {
						h.logger.Debugf("timeout for resource(%v) reached", d.resourceID.GetHref())
						h.updateAppliedConfigurationPendingResources(ctx, d.id, d.owner, d.resourceID.GetHref(), pb.AppliedDeviceConfiguration_Resource_TIMEOUT)
					}),
			)
		} else {
			resource.Status = pb.AppliedDeviceConfiguration_Resource_DONE
			resource.ResourceUpdated = &events.ResourceUpdated{
				ResourceId: resourceID,
				Status:     commands.Status_ERROR,
				Content: &commands.Content{
					Data:        []byte(errA.Error()),
					ContentType: message.TextPlain.String(),
				},
				AuditContext: &commands.AuditContext{
					// UserId:        owner,
					CorrelationId: resource.GetCorrelationId(),
					Owner:         owner,
				},
				EventMetadata: &events.EventMetadata{
					ConnectionId: confID,
				},
				// ResourceTypes: ,
				// OpenTelemetryCarrier:
			}
		}
		resources[href] = resource
	}

	appliedConf.Resources = maps.Values(resources)
	_, errU := h.storage.UpdateAppliedConfiguration(ctx, appliedConf)
	return errU
}

func EvalJQ(jq string, v any) (bool, error) {
	q, err := gojq.Parse(jq)
	if err != nil {
		return false, fmt.Errorf("cannot parse jq query(%v): %w", jq, err)
	}
	iter := q.Run(v)
	val, ok := iter.Next()
	if !ok {
		return false, nil
	}
	if result, ok := val.(bool); ok {
		return result, nil
	}
	return false, fmt.Errorf("invalid jq result: %v", val)
}

func (h *resourceChangedHandler) applyConfigurations(ctx context.Context, rc *events.ResourceChanged) error {
	owner := rc.GetAuditContext().GetOwner()
	if owner == "" {
		return errors.New("owner not set")
	}

	var rcData map[string]any
	eval := func(condition *pb.Condition) bool {
		jq := condition.GetJqExpressionFilter()
		if jq == "" {
			return true
		}
		if rcData == nil {
			if err := commands.DecodeContent(rc.GetContent(), &rcData); err != nil {
				h.logger.Errorf("cannot decode content: %w", err)
				return false
			}
		}
		ok, err := EvalJQ(jq, rcData)
		if err != nil {
			h.logger.Error(err)
			return false
		}
		return ok
	}

	resourceID := rc.GetResourceId()
	deviceID := resourceID.GetDeviceId()
	resourceHref := resourceID.GetHref()
	resourceTypes := rc.GetResourceTypes()
	// get matching conditions
	conditions, err := h.getConditions(ctx, owner, deviceID, resourceHref, resourceTypes, eval)
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
	return h.applyConfigurationToResources(ctx, owner, deviceID, confWithConditions)
}

func (h *resourceChangedHandler) finishPendingConfiguration(ctx context.Context, resourceID *commands.ResourceId) {
	pc, ok := h.pendingConfigurations.LoadAndDelete(resourceID.ToUUID())
	if !ok {
		return
	}
	pcd := pc.Data()
	h.logger.Debugf("resource(%v) updated", resourceID.GetHref())
	h.updateAppliedConfigurationPendingResources(ctx, pcd.id, pcd.owner, pcd.resourceID.GetHref(), pb.AppliedDeviceConfiguration_Resource_DONE)
}

func (h *resourceChangedHandler) Handle(ctx context.Context, iter eventbus.Iter) error {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var changed events.ResourceChanged
		if ev.EventType() == changed.EventType() {
			if err := ev.Unmarshal(&changed); err != nil {
				h.logger.Errorf("cannot unmarshal event: %w", err)
				continue
			}
			if err := h.applyConfigurations(ctx, &changed); err != nil {
				h.logger.Errorf("cannot apply configurations: %w", err)
			}
			continue
		}

		var updated events.ResourceUpdated
		if ev.EventType() == updated.EventType() {
			if err := ev.Unmarshal(&updated); err != nil {
				h.logger.Errorf("cannot unmarshal event: %w", err)
				continue
			}
			h.finishPendingConfiguration(ctx, updated.GetResourceId())
			continue
		}

		h.logger.Errorf("unexpected event type: %v", ev.EventType())
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
	subjectResourceChanged := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent,
		isEvents.WithOwner(owner),
		WithAllDevicesAndResources(),
		isEvents.WithEventType((&events.ResourceChanged{}).EventType()))
	subjectResourceUpdated := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent,
		isEvents.WithOwner(owner),
		WithAllDevicesAndResources(),
		isEvents.WithEventType((&events.ResourceUpdated{}).EventType()))
	observer, err := subscriber.Subscribe(ctx, subscriptionID, []string{subjectResourceChanged, subjectResourceUpdated}, handler)
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
