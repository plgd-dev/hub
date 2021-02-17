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

type resourceLinksCtx struct {
	lock      sync.Mutex
	deviceID  string
	resources map[string]*commands.Resource
	version   uint64

	subscriptions *subscriptions
}

func NewResourceLinksCtx(subscriptions *subscriptions) func(context.Context) eventstore.Model {
	return func(context.Context) eventstore.Model {
		return &resourceLinksCtx{
			subscriptions: subscriptions,
		}
	}
}

func (m *resourceLinksCtx) Clone() *resourceLinksCtx {
	m.lock.Lock()
	defer m.lock.Unlock()

	return &resourceLinksCtx{
		deviceID:  m.deviceID,
		resources: m.resources,
		version:   m.version,
	}
}

func (m *resourceLinksCtx) onResourcePublishedLocked(ctx context.Context) error {
	links := pb.RAResourcesToProto(m.resources)
	return m.subscriptions.OnResourceLinksPublished(ctx, m.deviceID, ResourceLinks{
		links:   links,
		version: m.version,
	})
}

func (m *resourceLinksCtx) onResourceUnpublishedLocked(ctx context.Context) error {
	links := pb.RAResourcesToProto(m.resources)
	return m.subscriptions.OnResourceLinksUnpublished(ctx, m.deviceID, ResourceLinks{
		links:   links,
		version: m.version,
	})
}

func (m *resourceLinksCtx) SnapshotEventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *resourceLinksCtx) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onResourcePublished, onResourceUnpublished bool
	m.lock.Lock()
	defer m.lock.Unlock()
	var anyEventProcessed bool
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		anyEventProcessed = true
		log.Debugf("grpc-gateway.resourceLinksCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupID(), eu.AggregateID(), eu.Version(), eu.EventType())
		m.version = eu.Version()
		switch eu.EventType() {
		case (&events.ResourceLinksSnapshotTaken{}).EventType():
			var s events.ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			m.deviceID = s.GetDeviceId()
			m.resources = s.GetResources()
		case (&events.ResourceLinksPublished{}).EventType():
			var s events.ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			m.deviceID = s.GetDeviceId()
			for _, res := range s.GetResources() {
				m.resources[res.GetHref()] = res
			}
			onResourcePublished = true
		case (&events.ResourceLinksUnpublished{}).EventType():
			var s events.ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}

			m.deviceID = s.GetDeviceId()
			if len(m.resources) == len(s.GetResources()) {
				m.resources = make(map[string]*commands.Resource)
			} else {
				for _, res := range s.GetResources() {
					delete(m.resources, res.GetHref())
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
		if err := m.onResourcePublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	} else if onResourceUnpublished {
		if err := m.onResourceUnpublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	return nil
}
