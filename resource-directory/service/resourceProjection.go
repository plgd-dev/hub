package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type resourceProjection struct {
	lock                     sync.Mutex
	resourceID               *commands.ResourceId
	content                  *events.ResourceChanged
	version                  uint64
	onResourceChangedVersion uint64

	resourceUpdatePendings   []*events.ResourceUpdatePending
	resourceRetrievePendings []*events.ResourceRetrievePending
	resourceDeletePendings   []*events.ResourceDeletePending
	resourceCreatePendings   []*events.ResourceCreatePending
}

func NewResourceProjection() eventstore.Model {
	return &resourceProjection{
		resourceUpdatePendings:   make([]*events.ResourceUpdatePending, 0, 8),
		resourceRetrievePendings: make([]*events.ResourceRetrievePending, 0, 8),
		resourceDeletePendings:   make([]*events.ResourceDeletePending, 0, 8),
		resourceCreatePendings:   make([]*events.ResourceCreatePending, 0, 8),
	}
}

func (rp *resourceProjection) cloneLocked() *resourceProjection {
	resourceCreatePendings := make([]*events.ResourceCreatePending, 0, len(rp.resourceCreatePendings))
	resourceCreatePendings = append(resourceCreatePendings, rp.resourceCreatePendings...)
	resourceRetrievePendings := make([]*events.ResourceRetrievePending, 0, len(rp.resourceRetrievePendings))
	resourceRetrievePendings = append(resourceRetrievePendings, rp.resourceRetrievePendings...)
	resourceUpdatePendings := make([]*events.ResourceUpdatePending, 0, len(rp.resourceUpdatePendings))
	resourceUpdatePendings = append(resourceUpdatePendings, rp.resourceUpdatePendings...)
	resourceDeletePendings := make([]*events.ResourceDeletePending, 0, len(rp.resourceDeletePendings))
	resourceDeletePendings = append(resourceDeletePendings, rp.resourceDeletePendings...)
	return &resourceProjection{
		resourceID:               rp.resourceID,
		content:                  rp.content,
		version:                  rp.version,
		resourceUpdatePendings:   resourceUpdatePendings,
		resourceCreatePendings:   resourceCreatePendings,
		resourceRetrievePendings: resourceRetrievePendings,
		resourceDeletePendings:   resourceDeletePendings,
	}
}

func (rp *resourceProjection) Clone() *resourceProjection {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	return rp.cloneLocked()
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
	rp.resourceID = s.ResourceId
	rp.content = s.LatestResourceChange
	rp.onResourceChangedVersion = eu.Version()
	rp.resourceUpdatePendings = s.GetResourceUpdatePendings()
	rp.resourceCreatePendings = s.GetResourceCreatePendings()
	rp.resourceDeletePendings = s.GetResourceDeletePendings()
	rp.resourceRetrievePendings = s.GetResourceRetrievePendings()
	return nil
}

func (rp *resourceProjection) handleResourceChangedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceChanged
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	rp.content = &s
	rp.onResourceChangedVersion = eu.Version()
	return nil
}

func (rp *resourceProjection) handleResourceUpdatePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceUpdatePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceUpdatePendings = append(rp.resourceUpdatePendings, &s)
	rp.resourceID = s.ResourceId
	return nil
}

func (rp *resourceProjection) handleResourceUpdatedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceUpdated
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	tmp := make([]*events.ResourceUpdatePending, 0, 16)
	var found bool
	for _, cu := range rp.resourceUpdatePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.resourceUpdatePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceRetrievePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceRetrievePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	rp.resourceRetrievePendings = append(rp.resourceRetrievePendings, &s)
	return nil
}

func (rp *resourceProjection) handleResourceDeletePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceDeletePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	rp.resourceDeletePendings = append(rp.resourceDeletePendings, &s)
	return nil
}

func (rp *resourceProjection) handleResourceRetrievedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceRetrieved
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	tmp := make([]*events.ResourceRetrievePending, 0, 16)
	var found bool
	for _, cu := range rp.resourceRetrievePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.resourceRetrievePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceDeletedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceDeleted
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	tmp := make([]*events.ResourceDeletePending, 0, 16)
	var found bool
	for _, cu := range rp.resourceDeletePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.resourceDeletePendings = tmp
	}
	return nil
}

func (rp *resourceProjection) handleResourceCreatePendingLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceCreatePending
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceCreatePendings = append(rp.resourceCreatePendings, &s)
	rp.resourceID = s.ResourceId
	return nil
}

func (rp *resourceProjection) handleResourceCreatedLocked(eu eventstore.EventUnmarshaler) error {
	var s events.ResourceCreated
	if err := eu.Unmarshal(&s); err != nil {
		return err
	}
	rp.resourceID = s.ResourceId
	tmp := make([]*events.ResourceCreatePending, 0, 16)
	var found bool
	for _, cu := range rp.resourceCreatePendings {
		if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
			tmp = append(tmp, cu)
		} else {
			found = true
		}
	}
	if found {
		rp.resourceCreatePendings = tmp
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

	rp.lock.Lock()
	defer rp.lock.Unlock()
	var groupID, aggregateID string
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("resourceProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		groupID = eu.GroupID()
		aggregateID = eu.AggregateID()
		rp.version = eu.Version()

		handler, ok := eventTypeToRPHandler[eu.EventType()]
		if !ok {
			log.Debugf("unhandled event type %v", eu.EventType())
			continue
		}
		if err := handler(eu); err != nil {
			return err
		}
	}
	if rp.resourceID == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
	}
	return nil
}
