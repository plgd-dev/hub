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

	subscriptions *Subscriptions
}

func NewResourceLinksProjection(subscriptions *Subscriptions) eventstore.Model {
	return &resourceLinksProjection{
		resources:     make(map[string]*commands.Resource),
		subscriptions: subscriptions,
	}
}

func (rlp *resourceLinksProjection) Clone() *resourceLinksProjection {
	rlp.lock.Lock()
	defer rlp.lock.Unlock()
	resources := make(map[string]*commands.Resource)
	for href, resource := range rlp.resources {
		resources[href] = resource
	}

	return &resourceLinksProjection{
		deviceID:  rlp.deviceID,
		resources: resources,
		version:   rlp.version,
	}
}

func (rlp *resourceLinksProjection) InitialNotifyOfPublishedResourceLinks(ctx context.Context, subscription *deviceSubscription) error {
	rlp.lock.Lock()
	defer rlp.lock.Unlock()

	links := pb.RAResourcesToProto(rlp.resources)
	return subscription.NotifyOfPublishedResourceLinks(ctx, ResourceLinks{
		links:   links,
		version: rlp.version,
		isInit:  true,
	})
}

func (rlp *resourceLinksProjection) onResourcePublishedLocked(ctx context.Context, publishedResources map[string]*commands.Resource) error {
	links := pb.RAResourcesToProto(publishedResources)
	return rlp.subscriptions.OnResourceLinksPublished(ctx, rlp.deviceID, ResourceLinks{
		links:   links,
		version: rlp.version,
	})
}

func (rlp *resourceLinksProjection) onResourceUnpublishedLocked(ctx context.Context, unpublishedResources map[string]*commands.Resource) error {
	links := pb.RAResourcesToProto(unpublishedResources)
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
	publishedResources := make(map[string]*commands.Resource)
	unpublishedResources := make(map[string]*commands.Resource)
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
			var e events.ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}

			rlp.deviceID = e.GetDeviceId()
			rlp.resources = e.GetResources()
		case (&events.ResourceLinksPublished{}).EventType():
			var e events.ResourceLinksPublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}

			rlp.deviceID = e.GetDeviceId()
			for _, res := range e.GetResources() {
				rlp.resources[res.GetHref()] = res
				publishedResources[res.GetHref()] = res
				delete(unpublishedResources, res.GetHref())
			}
		case (&events.ResourceLinksUnpublished{}).EventType():
			var e events.ResourceLinksUnpublished
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}

			rlp.deviceID = e.GetDeviceId()
			if len(rlp.resources) == len(e.GetHrefs()) {
				rlp.resources = make(map[string]*commands.Resource)
				publishedResources = make(map[string]*commands.Resource)
			} else {
				for _, href := range e.GetHrefs() {
					unpublishedResources[href] = rlp.resources[href]
					delete(rlp.resources, href)
					delete(publishedResources, href)
				}
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if len(publishedResources) != 0 {
		if err := rlp.onResourcePublishedLocked(ctx, publishedResources); err != nil {
			log.Errorf("%v", err)
		}
	}
	if len(unpublishedResources) != 0 {
		if err := rlp.onResourceUnpublishedLocked(ctx, unpublishedResources); err != nil {
			log.Errorf("%v", err)
		}
	}

	return nil
}
