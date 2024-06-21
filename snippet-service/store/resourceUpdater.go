package store

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
)

type pendingConfiguration struct {
	id            string
	owner         string
	correlationID string
	resourceID    *commands.ResourceId
}

type ResourceUpdater struct {
	storage               Store
	raConn                *grpcClient.Client
	raClient              raService.ResourceAggregateClient
	pendingConfigurations *cache.Cache[string, *pendingConfiguration]
	logger                log.Logger
}

func newPendingConfigurationsCache(ctx context.Context, interval time.Duration) *cache.Cache[string, *pendingConfiguration] {
	c := cache.NewCache[string, *pendingConfiguration]()
	add := periodic.New(ctx.Done(), interval)
	add(func(now time.Time) bool {
		c.CheckExpirations(now)
		return true
	})
	return c
}

func NewResourceUpdater(ctx context.Context, config grpcClient.Config, pendingCommandsCheckInterval time.Duration, storage Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*ResourceUpdater, error) {
	raConn, err := grpcClient.New(config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	return &ResourceUpdater{
		storage:               storage,
		raConn:                raConn,
		raClient:              raService.NewResourceAggregateClient(raConn.GRPC()),
		pendingConfigurations: newPendingConfigurationsCache(ctx, pendingCommandsCheckInterval),
		logger:                logger,
	}, nil
}

type evaluateCondition = func(condition *pb.Condition) bool

func (h *ResourceUpdater) getConditions(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string, eval evaluateCondition) ([]*pb.Condition, error) {
	conditions := make([]*pb.Condition, 0, 4)
	err := h.storage.GetLatestEnabledConditions(ctx, owner, &GetLatestConditionsQuery{
		DeviceID:           deviceID,
		ResourceHref:       resourceHref,
		ResourceTypeFilter: resourceTypes,
	}, func(v *Condition) error {
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

func (h *ResourceUpdater) getConfigurations(ctx context.Context, owner string, conditions []*pb.Condition) ([]configurationWithConditions, error) {
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

	h.logger.Debugf("getting configurations for conditions: %v", idFilter)

	// get configurations
	configurations := make([]*pb.Configuration, 0, 4)
	err := h.storage.GetConfigurations(ctx, owner, &pb.GetConfigurationsRequest{
		IdFilter: idFilter,
	}, func(v *Configuration) error {
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

	ids := make([]string, 0, len(configurations))
	for _, c := range configurations {
		ids = append(ids, c.GetId())
	}
	h.logger.Debugf("got configurations for conditions: %v", ids)

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
		condIDs := make([]string, 0, len(confConditions))
		for _, cond := range confConditions {
			condIDs = append(condIDs, cond.GetId())
		}
		h.logger.Debugf("found %v conditions for configuration(id:%v)", condIDs, c.GetId())
	}

	return confsWithConditions, nil
}

type appliedCondition struct {
	id      string
	version uint64
	token   string
}

func (h *ResourceUpdater) applyConfigurationToResource(ctx context.Context, resourceID *commands.ResourceId, configurationID, correlationID string, cr *pb.Configuration_Resource, token string) (int64, error) {
	h.logger.Debugf("applying configuration(id:%v) to resource(%v)", configurationID, resourceID.GetHref())
	upd := &commands.UpdateResourceRequest{
		ResourceId:    resourceID,
		CorrelationId: correlationID,
		Content:       cr.GetContent(),
		TimeToLive:    cr.GetTimeToLive(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: configurationID,
		},
		Force: true,
	}
	if token != "" {
		ctx = pkgGrpc.CtxWithToken(ctx, token)
	}
	res, err := h.raClient.UpdateResource(ctx, upd)
	if err != nil {
		return 0, err
	}
	h.logger.Infof("configuration(id:%v) applied to resource(%v)", configurationID, resourceID.GetHref())
	return res.GetValidUntil(), nil
}

func (h *ResourceUpdater) findTokenAndApplyConfigurationToResource(ctx context.Context, resourceID *commands.ResourceId, configurationID, correlationID string, cr *pb.Configuration_Resource, conditions []*pb.Condition) (int64, appliedCondition, error) {
	for _, cond := range conditions {
		condID := cond.GetId()
		token := cond.GetApiAccessToken()
		validUntil, err := h.applyConfigurationToResource(ctx, resourceID, configurationID, correlationID, cr, token)
		if err != nil {
			if grpcCode := pkgGrpc.ErrToStatus(err).Code(); grpcCode == codes.Unauthenticated {
				h.logger.Debugf("cannot apply configuration(id:%v) to resource(%v): invalid token", configurationID, resourceID.GetHref())
				continue
			}
			h.logger.Errorf("cannot apply configuration(id:%v) to resource(%v): %w", configurationID, resourceID.GetHref(), err)
			return 0, appliedCondition{}, err
		}
		return validUntil, appliedCondition{id: condID, version: cond.GetVersion(), token: token}, nil
	}
	return 0, appliedCondition{}, errors.New("cannot apply configuration: no valid token found")
}

func (h *ResourceUpdater) timeoutAppliedConfigurationPendingResource(ctx context.Context, pd *pendingConfiguration) error {
	return h.storage.UpdateAppliedConfigurationPendingResource(ctx, pd.owner, UpdateAppliedConfigurationPendingResourceRequest{
		AppliedConfigurationID: pd.id,
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:          pd.resourceID.GetHref(),
			CorrelationId: pd.correlationID,
			Status:        pb.AppliedDeviceConfiguration_Resource_TIMEOUT,
			ResourceUpdated: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{
					DeviceId: pd.resourceID.GetDeviceId(),
					Href:     pd.resourceID.GetHref(),
				},
				Status: commands.Status_ERROR,
			},
		},
	})
}

func correlationID(configurationID, correlationID string) string {
	return configurationID + "." + correlationID
}

func (h *ResourceUpdater) applyConfigurationToResources(ctx context.Context, owner, deviceID string, confWithConditions *configurationWithConditions) error {
	resources := map[string]*pb.AppliedDeviceConfiguration_Resource{}
	for _, cr := range confWithConditions.configuration.GetResources() {
		resources[cr.GetHref()] = &pb.AppliedDeviceConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: uuid.NewString(),
			Status:        pb.AppliedDeviceConfiguration_Resource_QUEUED,
		}
	}

	appliedConf, errC := h.storage.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Owner:           owner,
		DeviceId:        deviceID,
		ConfigurationId: pb.MakeRelationTo(confWithConditions.configuration.GetId(), confWithConditions.configuration.GetVersion()),
		ExecutedBy:      pb.MakeExecutedByConditionId(confWithConditions.conditions[0].GetId(), confWithConditions.conditions[0].GetVersion()),
		Resources:       maps.Values(resources),
		Timestamp:       time.Now().UnixNano(),
	})
	if errC != nil {
		if IsDuplicateKeyError(errC) {
			// applied configuration already exists
			h.logger.Debugf("applied configuration already exists for device(%s) and configuration(%s): %v", deviceID,
				confWithConditions.configuration.GetId(), errC)
			return nil
		}
		return fmt.Errorf("cannot create applied device configuration: %w", errC)
	}

	var appliedCond appliedCondition
	for _, cr := range confWithConditions.configuration.GetResources() {
		href := cr.GetHref()
		resourceID := &commands.ResourceId{Href: href, DeviceId: deviceID}
		confID := confWithConditions.configuration.GetId()
		correlationID := correlationID(appliedConf.GetId(), uuid.NewString())
		var validUntil int64
		var errA error
		if appliedCond.id != "" {
			validUntil, errA = h.applyConfigurationToResource(ctx, resourceID, confID, correlationID, cr, appliedCond.token)
		} else {
			validUntil, appliedCond, errA = h.findTokenAndApplyConfigurationToResource(ctx, resourceID, confID, correlationID, cr, confWithConditions.conditions)
			if errA == nil {
				appliedConf.ExecutedBy = pb.MakeExecutedByConditionId(appliedCond.id, appliedCond.version)
			}
		}

		resource := resources[href]
		if errA == nil {
			resource.Status = pb.AppliedDeviceConfiguration_Resource_PENDING
			resource.ValidUntil = validUntil
			if validUntil > 0 {
				h.pendingConfigurations.Store(
					correlationID, cache.NewElement(
						&pendingConfiguration{
							id:            appliedConf.GetId(),
							owner:         owner,
							correlationID: correlationID,
							resourceID:    resourceID,
						},
						pkgTime.Unix(0, validUntil),
						func(d *pendingConfiguration) {
							h.logger.Infof("timeout for pending resource(%v) update reached", d.resourceID.GetHref())
							if errT := h.timeoutAppliedConfigurationPendingResource(ctx, d); errT != nil {
								h.logger.Errorf("failed to timeout pending applied configuration for resource(%v): %w", d.resourceID.GetHref(), errT)
							}
						}),
				)
			}
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
			}
		}
		resources[href] = resource
	}

	appliedConf.Resources = maps.Values(resources)
	_, errU := h.storage.UpdateAppliedConfiguration(ctx, appliedConf)
	return errU
}

func decodeContent(content *commands.Content) (interface{}, error) {
	var rcData map[string]any
	err := commands.DecodeContent(content, &rcData)
	if err == nil {
		return rcData, nil
	}
	// content could be a single value or an array
	var rcData2 interface{}
	err = commands.DecodeContent(content, &rcData2)
	if err == nil {
		return rcData2, nil
	}
	return nil, fmt.Errorf("cannot decode content: %w", err)
}

func (h *ResourceUpdater) applyConfigurationsByConditions(ctx context.Context, rc *events.ResourceChanged) error {
	owner := rc.GetAuditContext().GetOwner()
	if owner == "" {
		return errors.New("owner not set")
	}

	rcData, err := decodeContent(rc.GetContent())
	if err != nil {
		return err
	}

	eval := func(condition *pb.Condition) bool {
		jq := condition.GetJqExpressionFilter()
		if jq == "" {
			return true
		}
		ok, errE := EvalJQCondition(jq, rcData)
		if errE != nil {
			h.logger.Error(errE)
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
	h.logger.Debugf("found %v conditions for resource changed event(deviceID:%v, href:%v, resourceTypes %v)", len(conditions), deviceID, resourceHref, resourceTypes)

	// get configurations with tokens
	confsWithConditions, err := h.getConfigurations(ctx, owner, conditions)
	if err != nil {
		return err
	}
	if len(confsWithConditions) == 0 {
		return nil
	}

	// apply configurations to resources
	var errs *multierror.Error
	for _, c := range confsWithConditions {
		if len(c.configuration.GetResources()) == 0 {
			h.logger.Debugf("no resources found for configuration(id:%v) for device %s", c.configuration.GetId(), deviceID)
			continue
		}
		err2 := h.applyConfigurationToResources(ctx, owner, deviceID, &c)
		if err2 != nil {
			errs = multierror.Append(errs, err2)
		}
	}
	return errs.ErrorOrNil()
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func (h *ResourceUpdater) finishPendingConfiguration(ctx context.Context, updated *events.ResourceUpdated) error {
	correlationID := updated.GetAuditContext().GetCorrelationId()
	// correlationID from snippet-service is in the form of "appliedConfigurationID.correlationID"
	parts := strings.Split(correlationID, ".")
	if len(parts) < 2 || !isValidUUID(parts[0]) {
		return nil
	}
	pc, ok := h.pendingConfigurations.LoadAndDelete(correlationID)
	if ok {
		h.logger.Debugf("pending configuration(%v) for resource(%v:%v) update expiration handler removed", pc.Data().id, updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref())
	}
	owner := updated.GetAuditContext().GetOwner()
	return h.storage.UpdateAppliedConfigurationPendingResource(ctx, owner, UpdateAppliedConfigurationPendingResourceRequest{
		AppliedConfigurationID: parts[0],
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:            updated.GetResourceId().GetHref(),
			CorrelationId:   correlationID,
			Status:          pb.AppliedDeviceConfiguration_Resource_DONE,
			ResourceUpdated: updated,
		},
	})
}

func (h *ResourceUpdater) handleResourceChanged(ctx context.Context, ev eventbus.EventUnmarshaler) error {
	var changed events.ResourceChanged
	if err := ev.Unmarshal(&changed); err != nil {
		return fmt.Errorf("cannot unmarshal ResourceChanged event: %w", err)
	}
	if err := h.applyConfigurationsByConditions(ctx, &changed); err != nil {
		return fmt.Errorf("cannot apply configurations for event (deviceID: %v, href: %v, resourceTypes: %v): %w", changed.GetResourceId().GetDeviceId(), changed.GetResourceId().GetHref(), changed.GetResourceTypes(), err)
	}
	return nil
}

func (h *ResourceUpdater) handleResourceUpdated(ctx context.Context, ev eventbus.EventUnmarshaler) error {
	var updated events.ResourceUpdated
	if err := ev.Unmarshal(&updated); err != nil {
		return fmt.Errorf("cannot unmarshal ResourceUpdated event: %w", err)
	}
	if err := h.finishPendingConfiguration(ctx, &updated); err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("failed to finish pending applied configuration for resource(%v): %w", updated.GetResourceId().GetHref(), err)
	}
	return nil
}

func (h *ResourceUpdater) Handle(ctx context.Context, iter eventbus.Iter) error {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		if ev.EventType() == (&events.ResourceChanged{}).EventType() {
			if err := h.handleResourceChanged(ctx, ev); err != nil {
				h.logger.Errorf("cannot handle resource changed event: %w", err)
			}
			continue
		}
		if ev.EventType() == (&events.ResourceUpdated{}).EventType() {
			if err := h.handleResourceUpdated(ctx, ev); err != nil {
				h.logger.Errorf("cannot handle resource updated event: %w", err)
			}
			continue
		}

		h.logger.Errorf("unexpected event type: %v", ev.EventType())
	}
}

/*
func (h *ResourceUpdater) applyConfigurationOnDemand(ctx context.Context, conf *pb.Configuration, owner, deviceID string) error {
	if len(conf.GetResources()) == 0 {
		h.logger.Debugf("no resources found for configuration(id:%v) for device %s", conf.GetId(), deviceID)
		return nil
	}
	resources := map[string]*pb.AppliedDeviceConfiguration_Resource{}
	for _, cr := range conf.GetResources() {
		resources[cr.GetHref()] = &pb.AppliedDeviceConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: uuid.NewString(),
			Status:        pb.AppliedDeviceConfiguration_Resource_QUEUED,
		}
	}

	// force
	// -> cancel pending commands
	//// -> get pending commands and cancel them
	//// -> remove from h.pendingConfigurations
	// FindOneAndDelete

	/// TODO: ak force tak upsert
	appliedConf, errC := h.storage.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Owner:           owner,
		DeviceId:        deviceID,
		ConfigurationId: pb.MakeRelationTo(conf.GetId(), conf.GetVersion()),
		ExecutedBy:      pb.MakeExecutedByOnDemand(),
		Resources:       maps.Values(resources),
		Timestamp:       time.Now().UnixNano(),
	})
	if errC != nil {
		if IsDuplicateKeyError(errC) {
			// applied configuration already exists
			h.logger.Debugf("applied configuration already exists for device(%s) and configuration(%s)", deviceID, conf.GetId())
			return nil
		}
		return fmt.Errorf("cannot create applied device configuration: %w", errC)
	}

	for _, cr := range conf.GetResources() {
		href := cr.GetHref()
		resourceID := &commands.ResourceId{Href: href, DeviceId: deviceID}
		confID := conf.GetId()
		correlationID := correlationID(appliedConf.GetId(), uuid.NewString())
		var validUntil int64
		var errA error
		validUntil, errA = h.applyConfigurationToResource(ctx, resourceID, confID, cr, "")

		resource := resources[href]
		if errA == nil {
			resource.Status = pb.AppliedDeviceConfiguration_Resource_PENDING
			h.pendingConfigurations.Store(
				resourceID.ToUUID(), cache.NewElement(
					&pendingConfiguration{
						id:         appliedConf.GetId(),
						owner:      owner,
						resourceID: resourceID,
					},
					pkgTime.Unix(0, validUntil),
					func(d *pendingConfiguration) {
						h.logger.Debugf("timeout for resource(%v) reached", d.resourceID.GetHref())
						h.updateAppliedConfigurationPendingResource(ctx, d.id, d.owner, d.resourceID.GetHref(), pb.AppliedDeviceConfiguration_Resource_TIMEOUT)
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
*/

func (h *ResourceUpdater) InvokeConfiguration(ctx context.Context, owner string, req *pb.InvokeConfigurationRequest) error {
	if err := ValidateInvokeConfigurationRequest(req); err != nil {
		return err
	}
	// find configuration
	var confs []*pb.Configuration
	err := h.storage.GetLatestConfigurationsByID(ctx, owner, []string{req.GetConfigurationId()}, func(v *Configuration) error {
		c, err := v.GetLatest()
		if err != nil {
			return err
		}
		confs = append(confs, c.Clone())
		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot get configuration: %w", err)
	}
	if len(confs) < 1 {
		return fmt.Errorf("configuration not found: %v", req.GetConfigurationId())
	}
	return nil
}

func (h *ResourceUpdater) Close() error {
	return h.raConn.Close()
}
