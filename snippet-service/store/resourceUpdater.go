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

type executeByType int

const (
	executeByTypeCondition executeByType = iota
	executeByTypeOnDemand
)

type appliedCondition struct {
	id      string
	version uint64
	token   string
}

type execution struct {
	condition  appliedCondition
	conditions []*pb.Condition
	executeBy  executeByType
}

type executionResult struct {
	validUntil int64
	condition  appliedCondition
	executedBy executeByType
	err        error
}

func (e execution) setExecutedBy(ac *pb.AppliedDeviceConfiguration) {
	if e.executeBy == executeByTypeOnDemand {
		ac.ExecutedBy = pb.MakeExecutedByOnDemand()
		return
	}
	if e.condition.id != "" {
		ac.ExecutedBy = pb.MakeExecutedByConditionId(e.condition.id, e.condition.version)
		return
	}
	firstCondition := e.conditions[0]
	ac.ExecutedBy = pb.MakeExecutedByConditionId(firstCondition.GetId(), firstCondition.GetVersion())
}

func (h *ResourceUpdater) applyExecution(ctx context.Context, execution execution, resourceID *commands.ResourceId, configurationID, correlationID string, cr *pb.Configuration_Resource) executionResult {
	if execution.executeBy == executeByTypeOnDemand {
		// TODO
		return executionResult{}
	}

	if execution.condition.id != "" {
		validUntil, err := h.applyConfigurationToResource(ctx, resourceID, configurationID, correlationID, cr, execution.condition.token)
		if err != nil {
			return executionResult{err: err}
		}
		return executionResult{
			validUntil: validUntil,
			condition:  execution.condition,
			executedBy: executeByTypeCondition,
		}
	}

	validUntil, appliedCond, err := h.findTokenAndApplyConfigurationToResource(ctx, resourceID, configurationID, correlationID, cr, execution.conditions)
	if err != nil {
		return executionResult{err: err}
	}
	return executionResult{
		validUntil: validUntil,
		condition:  appliedCond,
		executedBy: executeByTypeCondition,
	}
}

type configurationWithExecution struct {
	configuration *pb.Configuration
	execution     execution
}

func (h *ResourceUpdater) getConfigurations(ctx context.Context, owner string, conditions []*pb.Condition) ([]configurationWithExecution, error) {
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
		confConditions = append(confConditions, c.Clone())
		confsToConditions[confID] = confConditions
		idFilter = append(idFilter, &pb.IDFilter{
			Id: confID,
			Version: &pb.IDFilter_Latest{
				Latest: true,
			},
		})
	}
	if (len(idFilter)) == 0 {
		return []configurationWithExecution{}, nil
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

	confsWithConditions := make([]configurationWithExecution, 0, len(configurations))
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
		confsWithConditions = append(confsWithConditions, configurationWithExecution{
			configuration: c.Clone(),
			execution: execution{
				conditions: confConditions,
				executeBy:  executeByTypeCondition,
			},
		})
		condIDs := make([]string, 0, len(confConditions))
		for _, cond := range confConditions {
			condIDs = append(condIDs, cond.GetId())
		}
		h.logger.Debugf("found %v conditions for configuration(id:%v)", condIDs, c.GetId())
	}

	return confsWithConditions, nil
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
		h.logger.Errorf("failed to apply configuration(id:%v) to resource(%v): %w", configurationID, resourceID.GetHref(), err)
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
	_, err := h.storage.UpdateAppliedConfigurationResource(ctx, pd.owner, UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: pd.id,
		StatusFilter:           []pb.AppliedDeviceConfiguration_Resource_Status{pb.AppliedDeviceConfiguration_Resource_PENDING},
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
	return err
}

func resourceCorrelationID(ids ...string) string {
	cID := ""
	for _, id := range ids {
		if id == "" {
			continue
		}

		if cID != "" {
			cID += "."
		}
		cID += id
	}
	return cID
}

func getAppliedConfigurationResources(resources []*pb.Configuration_Resource, appliedConfID, correlationID string) ([]*pb.AppliedDeviceConfiguration_Resource, map[string]string) {
	updatedResources := make([]*pb.AppliedDeviceConfiguration_Resource, 0, len(resources))
	updatedResourceCorIDs := make(map[string]string)
	for _, cr := range resources {
		hrefCorrelationID := uuid.NewString()
		resCorrelationID := resourceCorrelationID(appliedConfID, hrefCorrelationID, correlationID)
		updatedResourceCorIDs[cr.GetHref()] = resCorrelationID
		updatedResources = append(updatedResources, &pb.AppliedDeviceConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: resCorrelationID,
			Status:        pb.AppliedDeviceConfiguration_Resource_QUEUED,
		})
	}
	return updatedResources, updatedResourceCorIDs
}

func getUpdateAppliedConfigurationResourceRequest(appliedConfID, confID, owner, correlationID string, resourceID *commands.ResourceId, execution executionResult, setExecutionConditionID bool) UpdateAppliedConfigurationResourceRequest {
	update := UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: appliedConfID,
		StatusFilter:           []pb.AppliedDeviceConfiguration_Resource_Status{pb.AppliedDeviceConfiguration_Resource_QUEUED},
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:          resourceID.GetHref(),
			CorrelationId: correlationID,
		},
	}
	if execution.err == nil {
		// update resource status from queued to pending
		update.Resource.Status = pb.AppliedDeviceConfiguration_Resource_PENDING
		update.Resource.ValidUntil = execution.validUntil
		if setExecutionConditionID {
			update.AppliedCondition = &pb.AppliedDeviceConfiguration_RelationTo{
				Id:      execution.condition.id,
				Version: execution.condition.version,
			}
		}
		return update
	}

	update.Resource.Status = pb.AppliedDeviceConfiguration_Resource_DONE
	update.Resource.ResourceUpdated = &events.ResourceUpdated{
		ResourceId: resourceID,
		Status:     commands.Status_ERROR,
		Content: &commands.Content{
			Data:        []byte(execution.err.Error()),
			ContentType: message.TextPlain.String(),
		},
		AuditContext: &commands.AuditContext{
			// UserId:        owner,
			CorrelationId: correlationID,
			Owner:         owner,
		},
		EventMetadata: &events.EventMetadata{
			ConnectionId: confID,
		},
	}
	return update
}

func (h *ResourceUpdater) scheduleAppliedConfigurationPendingResourceTimeout(ctx context.Context, pd *pendingConfiguration, validUntil int64) {
	h.logger.Debugf("timeout(%v) for pending resource(%v) update scheduled for %v", pd.correlationID, pd.resourceID.GetHref(), time.Unix(0, validUntil))
	h.pendingConfigurations.Store(
		pd.correlationID, cache.NewElement(
			pd,
			pkgTime.Unix(0, validUntil),
			func(d *pendingConfiguration) {
				h.logger.Debugf("timeout for pending resource(%v) update reached", d.resourceID.GetHref())
				if errT := h.timeoutAppliedConfigurationPendingResource(ctx, d); errT != nil {
					h.logger.Errorf("failed to timeout pending applied configuration for resource(%v): %w", d.resourceID.GetHref(), errT)
				}
			}),
	)
}

func (h *ResourceUpdater) applyConfigurationToResources(ctx context.Context, owner, deviceID, correlationID string, confWithExecution *configurationWithExecution) (*pb.AppliedDeviceConfiguration, error) {
	h.logger.Debugf("applying configuration(id:%v)", confWithExecution.configuration.GetId())
	appliedConfID := uuid.NewString()
	resources, resourcesCorIDs := getAppliedConfigurationResources(confWithExecution.configuration.GetResources(), appliedConfID, correlationID)
	create := &pb.AppliedDeviceConfiguration{
		Id:              appliedConfID,
		Owner:           owner,
		DeviceId:        deviceID,
		ConfigurationId: pb.MakeRelationTo(confWithExecution.configuration.GetId(), confWithExecution.configuration.GetVersion()),
		Resources:       resources,
		Timestamp:       time.Now().UnixNano(),
	}
	confWithExecution.execution.setExecutedBy(create)

	appliedConf, errC := h.storage.CreateAppliedConfiguration(ctx, create)
	if errC != nil {
		return nil, fmt.Errorf("cannot create applied device configuration: %w", errC)
	}
	h.logger.Debugf("applied configuration created: %v", appliedConf)

	var errs *multierror.Error
	for _, cr := range confWithExecution.configuration.GetResources() {
		href := cr.GetHref()
		resourceID := &commands.ResourceId{Href: href, DeviceId: deviceID}
		confID := confWithExecution.configuration.GetId()
		resCorrelationID := resourcesCorIDs[href]
		exRes := h.applyExecution(ctx, confWithExecution.execution, resourceID, confID, resCorrelationID, cr)
		updateExecutionConditionID := false
		if exRes.executedBy == executeByTypeCondition && exRes.condition.id != appliedConf.GetConditionId().GetId() {
			// update for next iteration
			confWithExecution.execution.condition = exRes.condition
			updateExecutionConditionID = true
		}
		update := getUpdateAppliedConfigurationResourceRequest(appliedConf.GetId(), confID, owner, resCorrelationID, resourceID, exRes, updateExecutionConditionID)
		h.logger.Debugf("updating applied configuration(%v) resource(%v) with status(%v)", appliedConf.GetId(), href, update.Resource.GetStatus().String())
		var err error
		// TODO: need to distinguish between:
		//  - if the appliedConfiguration doesnt exists -> it was removed by forced InvokeConfiguration -> exit
		//  - if the resource is not in queued status -> it was already updated by other goroutine -> skip
		appliedConf, err = h.storage.UpdateAppliedConfigurationResource(ctx, owner, update)
		if err != nil {
			h.logger.Errorf("cannot update applied configuration resource: %w", err)
			errs = multierror.Append(errs, err)
			continue
		}
		if exRes.validUntil > 0 {
			h.scheduleAppliedConfigurationPendingResourceTimeout(ctx, &pendingConfiguration{
				id:            appliedConf.GetId(),
				owner:         owner,
				correlationID: resCorrelationID,
				resourceID:    resourceID,
			}, exRes.validUntil)
		}
	}
	return appliedConf, errs.ErrorOrNil()
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
		_, errA := h.applyConfigurationToResources(ctx, owner, deviceID, "", &c)
		if IsDuplicateKeyError(errA) {
			// applied configuration already exists
			h.logger.Debugf("applied configuration already exists for device(%s) and configuration(%s): %v", deviceID,
				c.configuration.GetId(), errA)
			continue
		}
		if errA != nil {
			errs = multierror.Append(errs, errA)
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
	// correlationID from snippet-service is in the form of
	// 1) "appliedConfigurationID.resourceCorrelationID" if the configuration was applied by a condition
	// 2) "appliedConfigurationID.resourceCorrelationID{. InvokeConfiguration correlationID}" if the configuration was applied on demand by InvokeConfiguration
	parts := strings.Split(correlationID, ".")
	if len(parts) < 2 || !isValidUUID(parts[0]) {
		return nil
	}
	pc, ok := h.pendingConfigurations.LoadAndDelete(correlationID)
	if ok {
		h.logger.Debugf("pending configuration(%v) for resource(%v:%v) update expiration handler removed", pc.Data().id, updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref())
	}
	owner := updated.GetAuditContext().GetOwner()
	_, err := h.storage.UpdateAppliedConfigurationResource(ctx, owner, UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: parts[0],
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:            updated.GetResourceId().GetHref(),
			CorrelationId:   correlationID,
			Status:          pb.AppliedDeviceConfiguration_Resource_DONE,
			ResourceUpdated: updated,
		},
	})
	return err
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

func (h *ResourceUpdater) applyConfigurationOnDemand(ctx context.Context, conf *pb.Configuration, owner, deviceID, correlationID string, force bool) (*pb.AppliedDeviceConfiguration, error) {
	if len(conf.GetResources()) == 0 {
		h.logger.Debugf("no resources found for configuration(id:%v) for device %s", conf.GetId(), deviceID)
		return nil, nil
	}

	if force {
		// TODO
		// if force FindOneAndReplace + upsert=true
		// -> cancel pending commands
		//// -> get pending commands and cancel them
		//// -> remove from h.pendingConfigurations
		// FindOneAndReplace
		//// think about: what if the applied configuration is already in progress?
		return nil, nil
	}
	return h.applyConfigurationToResources(ctx, owner, deviceID, correlationID, &configurationWithExecution{
		configuration: conf,
		execution: execution{
			executeBy: executeByTypeOnDemand,
		},
	})
}

func (h *ResourceUpdater) InvokeConfiguration(ctx context.Context, owner string, req *pb.InvokeConfigurationRequest, p ProccessAppliedDeviceConfigurations) error {
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
	appliedConf, err := h.applyConfigurationOnDemand(ctx, confs[0], owner, req.GetDeviceId(), req.GetCorrelationId(), req.GetForce())
	if err != nil {
		return fmt.Errorf("cannot apply configuration: %w", err)
	}
	if appliedConf != nil {
		return p(appliedConf)
	}
	return nil
}

func (h *ResourceUpdater) Close() error {
	return h.raConn.Close()
}
