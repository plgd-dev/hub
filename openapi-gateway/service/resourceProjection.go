package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/ocf-cloud/openapi-connector/events"

	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/schema/cloud"

	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/http"

	raCqrs "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs"
	raEvents "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/events"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/notification"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
)

type resourceCtx struct {
	lock        sync.Mutex
	resource    *pbRA.Resource
	isPublished bool
	content     *pbRA.ResourceChanged

	syncPoolHandler             *GoroutinePoolHandler
	updateNotificationContainer *notification.UpdateNotificationContainer
}

func newResourceCtx(syncPoolHandler *GoroutinePoolHandler, updateNotificationContainer *notification.UpdateNotificationContainer) func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			syncPoolHandler:             syncPoolHandler,
			updateNotificationContainer: updateNotificationContainer,
		}, nil
	}
}

func (m *resourceCtx) cloneLocked() *resourceCtx {
	return &resourceCtx{
		resource:    m.resource,
		isPublished: m.isPublished,
		content:     m.content,
	}
}

func (m *resourceCtx) Clone() *resourceCtx {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.cloneLocked()
}

func (m *resourceCtx) onResourcePublishedLocked(ctx context.Context) error {
	err := m.syncPoolHandler.Handle(ctx, Event{
		Id:             m.resource.GetDeviceId(),
		EventType:      events.EventType_ResourcesPublished,
		DeviceID:       m.resource.GetDeviceId(),
		Href:           m.resource.GetHref(),
		Representation: makeLinksRepresentation(events.EventType_ResourcesPublished, []eventstore.Model{m.cloneLocked()}),
	})
	if err != nil {
		err = fmt.Errorf("cannot make action on resource published: %w", err)
	}
	return err
}

func (m *resourceCtx) onCloudStatusChangedLocked(ctx context.Context) error {
	var decoder func(data []byte, v interface{}) error
	switch m.content.GetContent().GetContentType() {
	case coap.AppCBOR.String(), coap.AppOcfCbor.String():
		decoder = cbor.Decode
	case coap.AppJSON.String():
		decoder = json.Decode
	}
	if decoder == nil {
		return fmt.Errorf("decoder not found")
	}
	var cloudStatus cloud.Status
	err := decoder(m.content.GetContent().GetData(), &cloudStatus)
	if err != nil {
		return err
	}
	eventType := events.EventType_DevicesOffline
	if cloudStatus.Online {
		eventType = events.EventType_DevicesOnline
	}

	return m.syncPoolHandler.Handle(ctx, Event{
		Id:             m.resource.GetDeviceId(),
		EventType:      eventType,
		DeviceID:       m.resource.GetDeviceId(),
		Representation: makeOnlineOfflineRepresentation(m.resource.GetDeviceId()),
	})
}

func (m *resourceCtx) onResourceUnpublishedLocked(ctx context.Context) error {
	err := m.syncPoolHandler.Handle(ctx, Event{
		Id:             m.resource.GetDeviceId(),
		EventType:      events.EventType_ResourcesUnpublished,
		DeviceID:       m.resource.GetDeviceId(),
		Href:           m.resource.GetHref(),
		Representation: makeLinksRepresentation(events.EventType_ResourcesUnpublished, []eventstore.Model{m.cloneLocked()}),
	})
	if err != nil {
		err = fmt.Errorf("cannot make action on resource unpublished: %w", err)
	}
	return err
}

func (m *resourceCtx) onResourceChangedLocked(ctx context.Context) error {
	var rep interface{}
	var err error
	eventType := events.EventType_ResourceChanged
	if m.content.GetStatus() != pbRA.Status_OK {
		eventType = events.EventType_SubscriptionCanceled
	} else {
		rep, err = unmarshalContent(m.content.GetContent())
		if err != nil {
			return fmt.Errorf("cannot make action on resource content changed: %w", err)
		}
	}

	err = m.syncPoolHandler.Handle(ctx, Event{
		Id:             m.resource.GetDeviceId(),
		EventType:      eventType,
		DeviceID:       m.resource.GetDeviceId(),
		Href:           m.resource.GetHref(),
		Representation: rep,
	})
	if err != nil {
		err = fmt.Errorf("cannot make action on resource content changed: %w", err)
	}
	return err
}

func (m *resourceCtx) onProcessedContentUpdatesLocked(updateProcessed []raEvents.ResourceUpdated) {
	for _, up := range updateProcessed {
		notify := m.updateNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send notification", m.resource.DeviceId, m.resource.Id)
			}
		}
	}
}

func (m *resourceCtx) SnapshotEventType() string {
	s := &raEvents.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *resourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler
	var onResourcePublished, onResourceUnpublished, onResourceChanged bool
	processedContentUpdates := make([]raEvents.ResourceUpdated, 0, 128)
	m.lock.Lock()
	defer m.lock.Unlock()
	var anyEventProcessed bool
	for iter.Next(ctx, &eu) {
		anyEventProcessed = true
		log.Debugf("resourceCtx.Handle: DeviceID: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupId, eu.AggregateId, eu.Version, eu.EventType)
		switch eu.EventType {
		case http.ProtobufContentType(&pbRA.ResourceStateSnapshotTaken{}):
			var s raEvents.ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if !m.isPublished {
				onResourcePublished = s.IsPublished
				onResourceUnpublished = !s.IsPublished
			}
			if m.content == nil {
				onResourceChanged = true
			} else {
				onResourceChanged = s.GetLatestResourceChange().GetEventMetadata().GetVersion() > m.content.GetEventMetadata().GetVersion()
			}
			m.content = s.LatestResourceChange
			m.resource = s.Resource
			m.isPublished = s.IsPublished
		case http.ProtobufContentType(&pbRA.ResourcePublished{}):
			var s raEvents.ResourcePublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if !m.isPublished {
				onResourcePublished = true
				onResourceUnpublished = false
			}
			m.isPublished = true
			m.resource = s.Resource
		case http.ProtobufContentType(&pbRA.ResourceUnpublished{}):
			if m.isPublished {
				onResourcePublished = false
				onResourceUnpublished = true
			}
			m.isPublished = false
		case http.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if m.content == nil {
				onResourceChanged = true
			} else {
				onResourceChanged = s.GetEventMetadata().GetVersion() > m.content.GetEventMetadata().GetVersion()
			}
			m.content = &s.ResourceChanged
		case http.ProtobufContentType(&pbRA.ResourceUpdated{}):
			var s raEvents.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			processedContentUpdates = append(processedContentUpdates, s)
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if m.resource == nil {
		return fmt.Errorf("DeviceID: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", eu.GroupId, eu.AggregateId)
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

	if onResourceChanged && m.isPublished {
		if raCqrs.MakeResourceId(m.resource.GetDeviceId(), cloud.StatusHref) == m.resource.Id {
			if err := m.onCloudStatusChangedLocked(ctx); err != nil {
				log.Errorf("cannot make action on cloud status changed: %v", err)
			}
		}

		if err := m.onResourceChangedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	m.onProcessedContentUpdatesLocked(processedContentUpdates)

	return nil
}
