package service

import (
	"context"
	"sync"

	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/resource-aggregate/events"
)

type resourceLinksProjection struct {
	lock     sync.Mutex
	snapshot *events.ResourceLinksSnapshotTaken
}

func NewResourceLinksProjection() eventstore.Model {
	return &resourceLinksProjection{}
}

func (rlp *resourceLinksProjection) Clone() *resourceLinksProjection {
	var snapshot *events.ResourceLinksSnapshotTaken
	rlp.lock.Lock()
	defer rlp.lock.Unlock()

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
	rlp.lock.Lock()
	defer rlp.lock.Unlock()
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
			if _, err := rlp.snapshot.HandleEventResourceLinksPublished(ctx, &e); err != nil {
				return err
			}
		case (&events.ResourceLinksUnpublished{}).EventType():
			var e events.ResourceLinksUnpublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			if _, err := rlp.snapshot.HandleEventResourceLinksUnpublished(ctx, nil, &e); err != nil {
				return err
			}
		}
	}
	return nil
}
