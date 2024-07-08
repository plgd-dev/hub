package updater

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/snippet-service/jq"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
)

type ResourceUpdater struct {
	ctx       context.Context
	storage   store.Store
	raConn    *grpcClient.Client
	raClient  raService.ResourceAggregateClient
	scheduler gocron.Scheduler
	logger    log.Logger
}

func NewResourceUpdater(ctx context.Context, config ResourceUpdaterConfig, storage store.Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*ResourceUpdater, error) {
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}

	ru := &ResourceUpdater{
		ctx:      ctx,
		storage:  storage,
		raConn:   raConn,
		raClient: raService.NewResourceAggregateClient(raConn.GRPC()),
		logger:   logger,
	}
	if config.CleanUpExpiredUpdates != "" {
		scheduler, err := NewExpiredUpdatesChecker(config.CleanUpExpiredUpdates, config.ExtendCronParserBySeconds, ru)
		if err != nil {
			return nil, fmt.Errorf("cannot create scheduler: %w", err)
		}
		ru.scheduler = scheduler
	}
	return ru, nil
}

type evaluateCondition = func(condition *pb.Condition) bool

func (h *ResourceUpdater) getConditions(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string, eval evaluateCondition) ([]*pb.Condition, error) {
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

func (h *ResourceUpdater) applyExecution(ctx context.Context, execution execution, resourceID *commands.ResourceId, configurationID, correlationID string, cr *pb.Configuration_Resource) executionResult {
	if execution.executeBy == executeByTypeOnDemand {
		validUntil, err := h.applyConfigurationToResource(ctx, resourceID, configurationID, correlationID, cr, execution.onDemand.token)
		if err != nil {
			return executionResult{err: err}
		}
		return executionResult{
			validUntil: validUntil,
			onDemand:   execution.onDemand,
			executedBy: executeByTypeOnDemand,
		}
	}

	if execution.executeBy == executeByTypeCondition {
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

func (h *ResourceUpdater) getConfigurationsByConditions(ctx context.Context, owner string, conditions []*pb.Condition) ([]configurationWithExecution, error) {
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
				executeBy:  executeByTypeFindCondition,
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

func (h *ResourceUpdater) getConfigurations(ctx context.Context, owner, deviceID, resourceHref string, resourceTypes []string, eval evaluateCondition) ([]configurationWithExecution, error) {
	// get matching conditions
	conditions, err := h.getConditions(ctx, owner, deviceID, resourceHref, resourceTypes, eval)
	if err != nil {
		return nil, err
	}
	h.logger.Debugf("found %v conditions for resource changed event(deviceID:%v, href:%v, resourceTypes %v)", len(conditions), deviceID, resourceHref, resourceTypes)

	// get configurations with tokens
	return h.getConfigurationsByConditions(ctx, owner, conditions)
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

// 1) "appliedConfigurationID.resourceCorrelationID" if the configuration was applied by a condition
// 2) "appliedConfigurationID.resourceCorrelationID{. InvokeConfiguration correlationID}" if the configuration was applied on demand by InvokeConfiguration
func SplitCorrelationID(correlationID string) (string, string, string, bool) {
	parts := strings.Split(correlationID, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", "", false
	}
	appliedConfID := parts[0]
	resourceCorrelationID := parts[1]
	customCorrelationID := ""
	if len(parts) > 2 {
		customCorrelationID = parts[2]
	}
	return appliedConfID, resourceCorrelationID, customCorrelationID, true
}

func getAppliedConfigurationResources(resources []*pb.Configuration_Resource, appliedConfID, correlationID string) ([]*pb.AppliedConfiguration_Resource, map[string]string) {
	updatedResources := make([]*pb.AppliedConfiguration_Resource, 0, len(resources))
	updatedResourceCorIDs := make(map[string]string)
	for _, cr := range resources {
		hrefCorrelationID := uuid.NewString()
		resCorrelationID := resourceCorrelationID(appliedConfID, hrefCorrelationID, correlationID)
		updatedResourceCorIDs[cr.GetHref()] = resCorrelationID
		updatedResources = append(updatedResources, &pb.AppliedConfiguration_Resource{
			Href:          cr.GetHref(),
			CorrelationId: resCorrelationID,
			Status:        pb.AppliedConfiguration_Resource_QUEUED,
		})
	}
	return updatedResources, updatedResourceCorIDs
}

func makeResourceUpdatedWithError(confID, owner, correlationID string, resourceID *commands.ResourceId, err error) *events.ResourceUpdated {
	return &events.ResourceUpdated{
		ResourceId: resourceID,
		Status:     commands.Status_ERROR,
		Content: &commands.Content{
			Data:        []byte(err.Error()),
			ContentType: message.TextPlain.String(),
		},
		AuditContext: &commands.AuditContext{
			CorrelationId: correlationID,
			Owner:         owner,
		},
		EventMetadata: &events.EventMetadata{
			ConnectionId: confID,
		},
	}
}

func getUpdateAppliedConfigurationResourceRequest(appliedConfID, confID, owner, correlationID string, resourceID *commands.ResourceId, execution executionResult, setExecutionConditionID bool) store.UpdateAppliedConfigurationResourceRequest {
	update := store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: appliedConfID,
		StatusFilter:           []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_QUEUED},
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          resourceID.GetHref(),
			CorrelationId: correlationID,
		},
	}
	if execution.err == nil {
		// update resource status from queued to pending
		update.Resource.Status = pb.AppliedConfiguration_Resource_PENDING
		update.Resource.ValidUntil = execution.validUntil
		if setExecutionConditionID {
			update.AppliedCondition = &pb.AppliedConfiguration_LinkedTo{
				Id:      execution.condition.id,
				Version: execution.condition.version,
			}
		}
		return update
	}

	update.Resource.Status = pb.AppliedConfiguration_Resource_DONE
	update.Resource.ResourceUpdated = makeResourceUpdatedWithError(confID, owner, correlationID, resourceID, execution.err)
	return update
}

func (h *ResourceUpdater) cancelPendingResourceUpdate(ctx context.Context, resourceID *commands.ResourceId, correlationID, configurationID string) error {
	cancelReq := &commands.CancelPendingCommandsRequest{
		ResourceId:          resourceID,
		CorrelationIdFilter: []string{correlationID},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: configurationID,
		},
	}
	_, err := h.raClient.CancelPendingCommands(ctx, cancelReq)
	if err != nil {
		h.logger.Debugf("failed to cancel pending resource(%v) update: %v", resourceID.ToString(), err)
	} else {
		h.logger.Debugf("pending resource(%v) update canceled", resourceID.ToString())
	}
	return err
}

func (h *ResourceUpdater) cancelPendingResourceUpdates(appliedConf *pb.AppliedConfiguration, token string) {
	h.logger.Debugf("canceling pending resource operations for replaced configuration(%s)", appliedConf.GetId())

	for _, res := range appliedConf.GetResources() {
		if res.GetStatus() != pb.AppliedConfiguration_Resource_PENDING {
			continue
		}
		// since we are using the Force flag in the UpdateResourceRequest, we need to cancel the pending commands,
		// otherwise commands for not existing resources will remain in the pending state forever
		resourceID := &commands.ResourceId{DeviceId: appliedConf.GetDeviceId(), Href: res.GetHref()}
		_ = h.cancelPendingResourceUpdate(pkgGrpc.CtxWithToken(h.ctx, token), resourceID, res.GetCorrelationId(), appliedConf.GetConfigurationId().GetId())
	}
}

func ctxWithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return pkgGrpc.CtxWithToken(ctx, token)
}

func (h *ResourceUpdater) applyConfigurationToResources(ctx context.Context, owner, deviceID, correlationID string, confWithExecution *configurationWithExecution) (*pb.AppliedConfiguration, error) {
	h.logger.Debugf("applying configuration(id:%v)", confWithExecution.configuration.GetId())
	appliedConfID := uuid.NewString()
	resources, resourcesCorIDs := getAppliedConfigurationResources(confWithExecution.configuration.GetResources(), appliedConfID, correlationID)
	create := &pb.AppliedConfiguration{
		Id:              appliedConfID,
		Owner:           owner,
		DeviceId:        deviceID,
		ConfigurationId: pb.MakeLinkedTo(confWithExecution.configuration.GetId(), confWithExecution.configuration.GetVersion()),
		Resources:       resources,
		Timestamp:       time.Now().UnixNano(),
	}
	confWithExecution.execution.setExecutedBy(create)

	appliedConf, oldAppliedConf, errC := h.storage.CreateAppliedConfiguration(ctx, create, confWithExecution.execution.force)
	if errC != nil {
		return nil, fmt.Errorf("cannot create applied device configuration: %w", errC)
	}
	if oldAppliedConf != nil {
		h.cancelPendingResourceUpdates(oldAppliedConf, confWithExecution.execution.token())
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
		if exRes.executedBy == executeByTypeCondition {
			// update for next iteration
			// first resources always iterates conditions and on success it returns the condition id,
			// the same condition id is used for the remaining resources
			confWithExecution.execution.setCondition(exRes.condition)
			if exRes.condition.id != appliedConf.GetConditionId().GetId() {
				updateExecutionConditionID = true
			}
		}
		update := getUpdateAppliedConfigurationResourceRequest(appliedConf.GetId(), confID, owner, resCorrelationID, resourceID, exRes, updateExecutionConditionID)
		h.logger.Debugf("updating applied configuration(%v) resource(%v) with status(%v)", appliedConf.GetId(), href, update.Resource.GetStatus().String())
		var err error
		updatedAppliedConf, err := h.storage.UpdateAppliedConfigurationResource(ctx, owner, update)
		if err == nil {
			appliedConf = updatedAppliedConf
			continue
		}
		if errors.Is(err, store.ErrNotFound) { // the appliedConfiguration doesnt exists -> it was removed by forced InvokeConfiguration from other thread
			_ = h.cancelPendingResourceUpdate(ctxWithToken(ctx, exRes.token()), resourceID, resCorrelationID, confID)
			return nil, err
		}
		// the resource is not in queued status -> it was already updated by other goroutine -> skip
		h.logger.Errorf("cannot update applied configuration resource: %w", err)
		errs = multierror.Append(errs, err)
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
		jqe := condition.GetJqExpressionFilter()
		if jqe == "" {
			return true
		}
		ok, errE := jq.EvalJQCondition(jqe, rcData)
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
	confsWithConditions, err := h.getConfigurations(ctx, owner, deviceID, resourceHref, resourceTypes, eval)
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
		if store.IsDuplicateKeyError(errA) {
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
	appliedConfID, resourcesCorrelationID, _, ok := SplitCorrelationID(correlationID)
	if !ok || !isValidUUID(appliedConfID) || !isValidUUID(resourcesCorrelationID) {
		return nil
	}
	h.logger.Debugf("finishing pending configuration(%v) update for resource(%v:%v): %v", appliedConfID, updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref(), updated)
	owner := updated.GetAuditContext().GetOwner()
	_, err := h.storage.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: appliedConfID,
		Resource: &pb.AppliedConfiguration_Resource{
			Href:            updated.GetResourceId().GetHref(),
			CorrelationId:   correlationID,
			Status:          pb.AppliedConfiguration_Resource_DONE,
			ResourceUpdated: updated,
		},
	})
	if updated.GetStatus() == commands.Status_CANCELED && errors.Is(err, store.ErrNotFound) {
		// the pending update was canceled by h.cancelPendingResourceUpdates and the configuration was removed
		return nil
	}
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
	if err := h.finishPendingConfiguration(ctx, &updated); err != nil && !errors.Is(err, store.ErrNotFound) {
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

func (h *ResourceUpdater) applyConfigurationOnDemand(ctx context.Context, conf *pb.Configuration, token, owner, deviceID, correlationID string, force bool) (*pb.AppliedConfiguration, error) {
	if len(conf.GetResources()) == 0 {
		h.logger.Debugf("no resources found for configuration(id:%v) for device %s", conf.GetId(), deviceID)
		return nil, nil
	}

	return h.applyConfigurationToResources(ctx, owner, deviceID, correlationID, &configurationWithExecution{
		configuration: conf,
		execution: execution{
			executeBy: executeByTypeOnDemand,
			onDemand: appliedOnDemand{
				token: token,
			},
			force: force,
		},
	})
}

func (h *ResourceUpdater) InvokeConfiguration(ctx context.Context, token, owner string, req *pb.InvokeConfigurationRequest) (*pb.AppliedConfiguration, error) {
	if err := store.ValidateInvokeConfigurationRequest(req); err != nil {
		return nil, err
	}
	// find configuration
	var confs []*pb.Configuration
	err := h.storage.GetLatestConfigurationsByID(ctx, owner, []string{req.GetConfigurationId()}, func(v *store.Configuration) error {
		c, err := v.GetLatest()
		if err != nil {
			return err
		}
		confs = append(confs, c.Clone())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get configuration: %w", err)
	}
	if len(confs) < 1 {
		return nil, fmt.Errorf("configuration not found: %v", req.GetConfigurationId())
	}
	appliedConf, err := h.applyConfigurationOnDemand(ctx, confs[0], token, owner, req.GetDeviceId(), req.GetCorrelationId(), req.GetForce())
	if err != nil {
		return nil, fmt.Errorf("cannot apply configuration: %w", err)
	}
	return appliedConf, nil
}

func (h *ResourceUpdater) timeoutAppliedConfigurationPendingResource(ctx context.Context, owner, appliedConfigurationID, correlationID string, resourceID *commands.ResourceId) {
	_, err := h.storage.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: appliedConfigurationID,
		StatusFilter:           []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_PENDING},
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          resourceID.GetHref(),
			CorrelationId: correlationID,
			Status:        pb.AppliedConfiguration_Resource_TIMEOUT,
			ResourceUpdated: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{
					DeviceId: resourceID.GetDeviceId(),
					Href:     resourceID.GetHref(),
				},
				Status: commands.Status_ERROR,
			},
		},
	})
	if err != nil {
		h.logger.Errorf("failed to timeout pending applied configuration for resource(%v): %w", resourceID.GetHref(), err)
	}
}

func (h *ResourceUpdater) TimeoutPendingResourceUpdates() {
	h.logger.Debug("checking pending resource updates for timeout")
	// get expired pending updates from the database
	var pendingUpdates []*store.AppliedConfiguration
	_, err := h.storage.GetPendingAppliedConfigurationResourceUpdates(h.ctx, true, func(ac *store.AppliedConfiguration) error {
		pendingUpdates = append(pendingUpdates, ac)
		return nil
	})
	if err != nil {
		h.logger.Errorf("cannot get expired pending resource updates: %v", err)
		return
	}
	if len(pendingUpdates) == 0 {
		return
	}

	ctx := h.ctx
	// cancel pending updates
	for _, ac := range pendingUpdates {
		for _, res := range ac.GetResources() {
			resourceID := &commands.ResourceId{DeviceId: ac.GetDeviceId(), Href: res.GetHref()}
			h.logger.Debugf("timeout for pending resource(%v) update reached", resourceID.GetHref())
			h.timeoutAppliedConfigurationPendingResource(ctx, ac.GetOwner(), ac.GetId(), res.GetCorrelationId(), resourceID)
		}
	}
}

func (h *ResourceUpdater) CancelPendingResourceUpdates(ctx context.Context) error {
	h.logger.Debug("canceling pending resource updates")
	var pendingUpdates []*store.AppliedConfiguration
	_, err := h.storage.GetPendingAppliedConfigurationResourceUpdates(h.ctx, false, func(ac *store.AppliedConfiguration) error {
		pendingUpdates = append(pendingUpdates, ac)
		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot get pending applied configurations: %w", err)
	}
	if len(pendingUpdates) == 0 {
		return nil
	}

	var errs *multierror.Error
	for _, ac := range pendingUpdates {
		for _, res := range ac.GetResources() {
			resourceID := &commands.ResourceId{DeviceId: ac.GetDeviceId(), Href: res.GetHref()}
			err = h.cancelPendingResourceUpdate(ctx, resourceID, res.GetCorrelationId(), ac.GetConfigurationId().GetId())
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}

func (h *ResourceUpdater) Close() error {
	var errs *multierror.Error
	if h.scheduler != nil {
		err := h.scheduler.Shutdown()
		errs = multierror.Append(errs, err)
	}
	err := h.raConn.Close()
	errs = multierror.Append(errs, err)
	return errs.ErrorOrNil()
}
