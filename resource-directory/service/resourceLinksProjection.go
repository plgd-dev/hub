package service

import (
	"context"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/kit/v2/strings"
)

// protected by lock in Projection struct in resource-aggregate/cqrs/eventstore/projection.go
type resourceLinksProjection struct {
	deviceID string

	private struct {
		lock     sync.RWMutex // protects snapshot
		snapshot *events.ResourceLinksSnapshotTaken
	}
}

func NewResourceLinksProjection(deviceID string) eventstore.Model {
	return &resourceLinksProjection{
		deviceID: deviceID,
	}
}

func (rlp *resourceLinksProjection) GetDeviceID() string {
	return rlp.deviceID
}

func (rlp *resourceLinksProjection) EventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.EventType()
}

func (rlp *resourceLinksProjection) IterateOverResources(onResource func(res *commands.Resource) (wantNext bool)) {
	rlp.private.lock.RLock()
	defer rlp.private.lock.RUnlock()

	for _, res := range rlp.private.snapshot.GetResources() {
		rlp.private.lock.RUnlock()
		wantNext := onResource(res)
		rlp.private.lock.RLock()
		if !wantNext {
			return
		}
	}
}
func (rlp *resourceLinksProjection) GetResource(href string) *commands.Resource {
	rlp.private.lock.RLock()
	defer rlp.private.lock.RUnlock()
	if rlp.private.snapshot.GetResources() == nil {
		return nil
	}
	return rlp.private.snapshot.GetResources()[href]
}

func (rlp *resourceLinksProjection) LenResources() int {
	rlp.private.lock.RLock()
	defer rlp.private.lock.RUnlock()
	return len(rlp.private.snapshot.GetResources())
}

func (rlp *resourceLinksProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	rlp.private.lock.Lock()
	defer rlp.private.lock.Unlock()
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("resourceLinksProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		if rlp.private.snapshot == nil {
			rlp.private.snapshot = events.NewResourceLinksSnapshotTaken()
		}
		eventMetadata := rlp.private.snapshot.GetEventMetadata().Clone()
		eventMetadata.Version = eu.Version()
		rlp.private.snapshot.EventMetadata = eventMetadata
		switch eu.EventType() {
		case (&events.ResourceLinksSnapshotTaken{}).EventType():
			var e events.ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.private.snapshot = &e
		case (&events.ResourceLinksPublished{}).EventType():
			var e events.ResourceLinksPublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.private.snapshot.HandleEventResourceLinksPublished(&e)
		case (&events.ResourceLinksUnpublished{}).EventType():
			var e events.ResourceLinksUnpublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.private.snapshot.HandleEventResourceLinksUnpublished(nil, &e)
		}
	}
	return nil
}

func (rlp *resourceLinksProjection) ToResourceLinksPublished(typeFilter strings.Set) *events.ResourceLinksPublished {
	resources := make([]*commands.Resource, 0, len(rlp.private.snapshot.GetResources()))
	rlp.private.lock.RLock()
	defer rlp.private.lock.RUnlock()
	for _, resource := range rlp.private.snapshot.GetResources() {
		if hasMatchingType(resource.ResourceTypes, typeFilter) {
			resources = append(resources, resource)
		}
	}
	if len(resources) == 0 {
		return nil
	}
	return &events.ResourceLinksPublished{
		DeviceId:      rlp.GetDeviceID(),
		EventMetadata: rlp.private.snapshot.GetEventMetadata(),
		Resources:     resources,
		AuditContext:  rlp.private.snapshot.GetAuditContext(),
	}
}
