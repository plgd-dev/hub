package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
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

func (rp *resourceProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	var groupID, aggregateID string
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		groupID = eu.GroupID()
		aggregateID = eu.AggregateID()
		rp.version = eu.Version()
		switch eu.EventType() {
		case (&events.ResourceStateSnapshotTaken{}).EventType():
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
		case (&events.ResourceChanged{}).EventType():
			var s events.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceID = s.ResourceId
			rp.content = &s
			rp.onResourceChangedVersion = eu.Version()
		case (&events.ResourceUpdatePending{}).EventType():
			var s events.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceUpdatePendings = append(rp.resourceUpdatePendings, &s)
			rp.resourceID = s.ResourceId
		case (&events.ResourceUpdated{}).EventType():
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
		case (&events.ResourceRetrievePending{}).EventType():
			var s events.ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceID = s.ResourceId
			rp.resourceRetrievePendings = append(rp.resourceRetrievePendings, &s)
		case (&events.ResourceDeletePending{}).EventType():
			var s events.ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceID = s.ResourceId
			rp.resourceDeletePendings = append(rp.resourceDeletePendings, &s)
		case (&events.ResourceRetrieved{}).EventType():
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
		case (&events.ResourceDeleted{}).EventType():
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
		case (&events.ResourceCreatePending{}).EventType():
			var s events.ResourceCreatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceCreatePendings = append(rp.resourceCreatePendings, &s)
			rp.resourceID = s.ResourceId
		case (&events.ResourceCreated{}).EventType():
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
		}
	}
	if rp.resourceID == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
	}
	return nil
}
