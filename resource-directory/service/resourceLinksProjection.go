package service

import (
	"context"
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/log"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

type resourceLinksProjection struct {
	lock      sync.Mutex
	deviceID  string
	resources map[string]*commands.Resource
	version   uint64

	subscriptions *subscriptions
}

func NewResourceLinksProjection(subscriptions *subscriptions) eventstore.Model {
	return &resourceLinksProjection{
		subscriptions: subscriptions,
	}
}

func (rlp *resourceLinksProjection) Clone() *resourceLinksProjection {
	rlp.lock.Lock()
	defer rlp.lock.Unlock()

	return &resourceLinksProjection{
		deviceID:  rlp.deviceID,
		resources: rlp.resources,
		version:   rlp.version,
	}
}

func (rlp *resourceLinksProjection) onResourcePublishedLocked(ctx context.Context) error {
	links := pb.RAResourcesToProto(rlp.resources)
	return rlp.subscriptions.OnResourceLinksPublished(ctx, rlp.deviceID, ResourceLinks{
		links:   links,
		version: rlp.version,
	})
}

func (rlp *resourceLinksProjection) onResourceUnpublishedLocked(ctx context.Context) error {
	links := pb.RAResourcesToProto(rlp.resources)
	return rlp.subscriptions.OnResourceLinksUnpublished(ctx, rlp.deviceID, ResourceLinks{
		links:   links,
		version: rlp.version,
	})
}

func (rlp *resourceLinksProjection) SnapshotEventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.SnapshotEventType()
}

func (rlp *resourceLinksProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onResourcePublished, onResourceUnpublished bool
	rlp.lock.Lock()
	defer rlp.lock.Unlock()
	var anyEventProcessed bool
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		anyEventProcessed = true
		rlp.version = eu.Version()
		switch eu.EventType() {
		case (&events.ResourceLinksSnapshotTaken{}).EventType():
			var s events.ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			rlp.deviceID = s.GetDeviceId()
			rlp.resources = s.GetResources()
		case (&events.ResourceLinksPublished{}).EventType():
			var s events.ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			rlp.deviceID = s.GetDeviceId()
			for _, res := range s.GetResources() {
				rlp.resources[res.GetHref()] = res
			}
			onResourcePublished = true
		case (&events.ResourceLinksUnpublished{}).EventType():
			var s events.ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			rlp.deviceID = s.GetDeviceId()
			if len(rlp.resources) == len(s.GetResources()) {
				rlp.resources = make(map[string]*commands.Resource)
			} else {
				for _, res := range s.GetResources() {
					delete(rlp.resources, res.GetHref())
				}
			}
			onResourceUnpublished = true
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if onResourcePublished {
		if err := rlp.onResourcePublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	} else if onResourceUnpublished {
		if err := rlp.onResourceUnpublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	return nil
}
