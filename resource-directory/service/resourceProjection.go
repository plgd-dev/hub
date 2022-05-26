package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// protected by lock in Projection struct in resource-aggregate/cqrs/eventstore/projection.go
type resourceProjection struct {
	private struct {
		lock                     sync.RWMutex // protects all fields
		resourceID               *commands.ResourceId
		content                  *events.ResourceChanged
		version                  uint64
		onResourceChangedVersion uint64
		resourceUpdatePendings   []*events.ResourceUpdatePending
		resourceRetrievePendings []*events.ResourceRetrievePending
		resourceDeletePendings   []*events.ResourceDeletePending
		resourceCreatePendings   []*events.ResourceCreatePending
	}
}

func NewResourceProjection() eventstore.Model {
	var p resourceProjection
	p.private.resourceUpdatePendings = make([]*events.ResourceUpdatePending, 0, 8)
	p.private.resourceRetrievePendings = make([]*events.ResourceRetrievePending, 0, 8)
	p.private.resourceDeletePendings = make([]*events.ResourceDeletePending, 0, 8)
	p.private.resourceCreatePendings = make([]*events.ResourceCreatePending, 0, 8)
	return &p
}

func (rp *resourceProjection) GetResourceChanged() *events.ResourceChanged {
	rp.private.lock.RLock()
	defer rp.private.lock.RUnlock()
	return rp.private.content
}

func (rp *resourceProjection) EventType() string {
	s := &events.ResourceStateSnapshotTaken{}
	return s.EventType()
}

func (rp *resourceProjection) handleResourceStateSnapshotTakenLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceStateSnapshotTaken
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	rp.private.content = s.LatestResourceChange
	rp.private.onResourceChangedVersion = eu.Version()
	rp.private.resourceUpdatePendings = s.GetResourceUpdatePendings()
	rp.private.resourceCreatePendings = s.GetResourceCreatePendings()
	rp.private.resourceDeletePendings = s.GetResourceDeletePendings()
	rp.private.resourceRetrievePendings = s.GetResourceRetrievePendings()
	return nil
}

func (rp *resourceProjection) handleResourceChangedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceChanged
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	rp.private.content = &s
	rp.private.onResourceChangedVersion = eu.Version()
	return nil
}

func (rp *resourceProjection) handleResourceUpdatePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceUpdatePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceUpdatePendings = append(rp.private.resourceUpdatePendings, &s)
	rp.private.resourceID = s.ResourceId
	return nil
}

func (rp *resourceProjection) handleResourceUpdatedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceUpdated
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	tmp := make([]*events.ResourceUpdatePending, 0, 16)
	var found bool
	for _, cu := range rp.private.resourceUpdatePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.private.resourceUpdatePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceRetrievePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceRetrievePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	rp.private.resourceRetrievePendings = append(rp.private.resourceRetrievePendings, &s)
	return nil
}

func (rp *resourceProjection) handleResourceDeletePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceDeletePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	rp.private.resourceDeletePendings = append(rp.private.resourceDeletePendings, &s)
	return nil
}

func (rp *resourceProjection) handleResourceRetrievedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceRetrieved
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	tmp := make([]*events.ResourceRetrievePending, 0, 16)
	var found bool
	for _, cu := range rp.private.resourceRetrievePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.private.resourceRetrievePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceDeletedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceDeleted
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	tmp := make([]*events.ResourceDeletePending, 0, 16)
	var found bool
	for _, cu := range rp.private.resourceDeletePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.private.resourceDeletePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceCreatePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceCreatePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceCreatePendings = append(rp.private.resourceCreatePendings, &s)
	rp.private.resourceID = s.ResourceId
	return nil
}

func (rp *resourceProjection) handleResourceCreatedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceCreated
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.private.resourceID = s.ResourceId
	tmp := make([]*events.ResourceCreatePending, 0, 16)
	var found bool
	for _, cu := range rp.private.resourceCreatePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.private.resourceCreatePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	type eventTypeHandler = func(eventstore.EventUnmarshaler) error
	eventTypeToRPHandler := map[string]eventTypeHandler{
		(&events.ResourceStateSnapshotTaken{}).EventType(): rp.handleResourceStateSnapshotTakenLocked,
		(&events.ResourceChanged{}).EventType():            rp.handleResourceChangedLocked,
		(&events.ResourceUpdatePending{}).EventType():      rp.handleResourceUpdatePendingLocked,
		(&events.ResourceUpdated{}).EventType():            rp.handleResourceUpdatedLocked,
		(&events.ResourceRetrievePending{}).EventType():    rp.handleResourceRetrievePendingLocked,
		(&events.ResourceRetrieved{}).EventType():          rp.handleResourceRetrievedLocked,
		(&events.ResourceDeletePending{}).EventType():      rp.handleResourceDeletePendingLocked,
		(&events.ResourceDeleted{}).EventType():            rp.handleResourceDeletedLocked,
		(&events.ResourceCreatePending{}).EventType():      rp.handleResourceCreatePendingLocked,
		(&events.ResourceCreated{}).EventType():            rp.handleResourceCreatedLocked,
	}

	rp.private.lock.Lock()
	defer rp.private.lock.Unlock()
	var groupID, aggregateID string
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("resourceProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		groupID = eu.GroupID()
		aggregateID = eu.AggregateID()
		rp.private.version = eu.Version()

		handler, ok := eventTypeToRPHandler[eu.EventType()]
		if !ok {
			log.Debugf("unhandled event type %v", eu.EventType())
			continue
		}
		if err := handler(eu); err != nil {
			return err
		}
	}
	if rp.private.resourceID == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
	}
	return nil
}

type resourcePending interface {
	*events.ResourceCreatePending | *events.ResourceRetrievePending | *events.ResourceUpdatePending | *events.ResourceDeletePending
	IsExpired(time.Time) bool
}

func appendPendingCmd[T resourcePending](pendingCmds []*pb.PendingCommand, commandFilter subscription.FilterBitmask, bit subscription.FilterBitmask, now time.Time, create func(v T) *pb.PendingCommand, data []T) []*pb.PendingCommand {
	if subscription.IsFilteredBit(commandFilter, bit) {
		for _, pendingCmd := range data {
			if pendingCmd.IsExpired(now) {
				continue
			}
			pendingCmds = append(pendingCmds, create(pendingCmd))
		}
	}
	return pendingCmds
}

func (rp *resourceProjection) ToPendingCommands(commandFilter subscription.FilterBitmask, now time.Time) []*pb.PendingCommand {
	pendingCmds := make([]*pb.PendingCommand, 0, 32)
	rp.private.lock.RLock()
	defer rp.private.lock.RUnlock()

	pendingCmds = appendPendingCmd(pendingCmds, commandFilter, subscription.FilterBitmaskResourceCreatePending, now, func(p *events.ResourceCreatePending) *pb.PendingCommand {
		return &pb.PendingCommand{
			Command: &pb.PendingCommand_ResourceCreatePending{
				ResourceCreatePending: p,
			},
		}
	}, rp.private.resourceCreatePendings)

	pendingCmds = appendPendingCmd(pendingCmds, commandFilter, subscription.FilterBitmaskResourceRetrievePending, now, func(p *events.ResourceRetrievePending) *pb.PendingCommand {
		return &pb.PendingCommand{
			Command: &pb.PendingCommand_ResourceRetrievePending{
				ResourceRetrievePending: p,
			},
		}
	}, rp.private.resourceRetrievePendings)

	pendingCmds = appendPendingCmd(pendingCmds, commandFilter, subscription.FilterBitmaskResourceUpdatePending, now, func(p *events.ResourceUpdatePending) *pb.PendingCommand {
		return &pb.PendingCommand{
			Command: &pb.PendingCommand_ResourceUpdatePending{
				ResourceUpdatePending: p,
			},
		}
	}, rp.private.resourceUpdatePendings)

	pendingCmds = appendPendingCmd(pendingCmds, commandFilter, subscription.FilterBitmaskResourceDeletePending, now, func(p *events.ResourceDeletePending) *pb.PendingCommand {
		return &pb.PendingCommand{
			Command: &pb.PendingCommand_ResourceDeletePending{
				ResourceDeletePending: p,
			},
		}
	}, rp.private.resourceDeletePendings)

	return pendingCmds
}
