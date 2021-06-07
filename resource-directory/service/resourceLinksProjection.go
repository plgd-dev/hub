package service

import (
	"context"
	"sync"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type resourceLinksProjection struct {
	lock          sync.Mutex
	snapshot      *events.ResourceLinksSnapshotTaken
	subscriptions *Subscriptions
}

func NewResourceLinksProjection(subscriptions *Subscriptions) eventstore.Model {
	return &resourceLinksProjection{
		subscriptions: subscriptions,
	}
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

func (rlp *resourceLinksProjection) InitialNotifyOfPublishedResourceLinks(ctx context.Context, subscription *subscription) error {
	rlp.lock.Lock()
	defer rlp.lock.Unlock()

	resources := make([]*commands.Resource, 0, len(rlp.snapshot.GetResources()))
	for _, r := range rlp.snapshot.GetResources() {
		resources = append(resources, r)
	}

	return subscription.NotifyOfPublishedResourceLinks(ctx, ResourceLinksPublished{
		data: &events.ResourceLinksPublished{
			Resources:     resources,
			DeviceId:      rlp.snapshot.GetDeviceId(),
			EventMetadata: rlp.snapshot.GetEventMetadata(),
		},
		isInit: true,
	})
}

func (rlp *resourceLinksProjection) onResourcePublishedLocked(ctx context.Context, publishedResources *events.ResourceLinksPublished) error {
	log.Debugf("resourceLinksProjection.onResourcePublishedLocked %v", publishedResources)
	return rlp.subscriptions.OnResourceLinksPublished(ctx, ResourceLinksPublished{
		data: publishedResources,
	})
}

func (rlp *resourceLinksProjection) onResourceUnpublishedLocked(ctx context.Context, unpublishedResources *events.ResourceLinksUnpublished) error {
	log.Debugf("resourceLinksProjection.onResourceUnpublishedLocked %v", unpublishedResources)
	return rlp.subscriptions.OnResourceLinksUnpublished(ctx, unpublishedResources)
}

func (rlp *resourceLinksProjection) EventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.EventType()
}

func (rlp *resourceLinksProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	sendEvents := make([]interface{}, 0, 4)
	rlp.lock.Lock()
	defer rlp.lock.Unlock()
	var anyEventProcessed bool
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("resourceLinksProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		anyEventProcessed = true
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
			sendEvents = append(sendEvents[:0], &e)
		case (&events.ResourceLinksPublished{}).EventType():
			var e events.ResourceLinksPublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			err := rlp.snapshot.HandleEventResourceLinksPublished(ctx, &e)
			if err != nil {
				return err
			}
			sendEvents = append(sendEvents, &e)
		case (&events.ResourceLinksUnpublished{}).EventType():
			var e events.ResourceLinksUnpublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			unpublished, err := rlp.snapshot.HandleEventResourceLinksUnpublished(ctx, &e)
			if err != nil {
				return err
			}
			if len(unpublished) > 0 {
				sendEvents = append(sendEvents, &e)
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	for _, e := range sendEvents {
		switch v := e.(type) {
		// TODO handle snapshot
		case *events.ResourceLinksPublished:
			if err := rlp.onResourcePublishedLocked(ctx, v); err != nil {
				log.Errorf("%v", err)
			}
		case *events.ResourceLinksUnpublished:
			if err := rlp.onResourceUnpublishedLocked(ctx, v); err != nil {
				log.Errorf("%v", err)
			}
		}
	}

	return nil
}
