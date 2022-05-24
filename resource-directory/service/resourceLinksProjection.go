package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// protected by lock in Projection struct in resource-aggregate/cqrs/eventstore/projection.go
type resourceLinksProjection struct {
	snapshot *events.ResourceLinksSnapshotTaken
}

func NewResourceLinksProjection() eventstore.Model {
	return &resourceLinksProjection{}
}

func (rlp *resourceLinksProjection) GetDeviceID() string {
	if rlp.snapshot == nil {
		return ""
	}
	return rlp.snapshot.GetDeviceId()
}

func (rlp *resourceLinksProjection) Clone() *resourceLinksProjection {
	var snapshot *events.ResourceLinksSnapshotTaken

	if rlp.snapshot != nil {
		s, _ := rlp.snapshot.TakeSnapshot(rlp.snapshot.Version())
		snapshot = s.(*events.ResourceLinksSnapshotTaken)
	}

	return &resourceLinksProjection{
		snapshot: snapshot,
	}
}

func (rlp *resourceLinksProjection) EventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.EventType()
}

func (rlp *resourceLinksProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("resourceLinksProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		if rlp.snapshot == nil {
			rlp.snapshot = events.NewResourceLinksSnapshotTaken()
		}
		rlp.snapshot.GetEventMetadata().Version = eu.Version()
		switch eu.EventType() {
		case (&events.ResourceLinksSnapshotTaken{}).EventType():
			var e events.ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.snapshot = &e
		case (&events.ResourceLinksPublished{}).EventType():
			var e events.ResourceLinksPublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.snapshot.HandleEventResourceLinksPublished(&e)
		case (&events.ResourceLinksUnpublished{}).EventType():
			var e events.ResourceLinksUnpublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			rlp.snapshot.HandleEventResourceLinksUnpublished(nil, &e)
		}
	}
	return nil
}
