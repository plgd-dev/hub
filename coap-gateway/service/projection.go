package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/http"

	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

type resourceCtx struct {
	lock                     sync.Mutex
	resource                 *pbRA.Resource
	isPublished              bool
	isProcessed              bool
	resourceUpdatePendings   []raEvents.ResourceUpdatePending
	resourceRetrievePendings []raEvents.ResourceRetrievePending
	contentCtx               *pbRA.ResourceChanged

	server *Server
}

func newResourceCtx(server *Server) func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			resourceUpdatePendings: make([]raEvents.ResourceUpdatePending, 0, 8),
			server:                 server,
		}, nil
	}
}

func (m *resourceCtx) Resource() *pbRA.Resource {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.resource
}

func (m *resourceCtx) onResourcePublishedLocked() {
	client := m.server.clientContainerByDeviceID.Find(m.resource.DeviceId)
	if client == nil {
		return
	}
	err := client.observeResource(context.Background(), m.resource, true)
	if err != nil {
		log.Errorf("cannot observe resource: %v", err)
	}
}

func (m *resourceCtx) onResourceUnpublishedLocked() {
	client := m.server.clientContainerByDeviceID.Find(m.resource.DeviceId)
	if client == nil {
		return
	}
	client.unobserveResources(client.coapConn.Context(), []*pbRA.Resource{m.resource}, map[string]bool{m.resource.Id: true})
}

func (m *resourceCtx) onResourceChangedLocked() {
	for _, obs := range m.server.observeResourceContainer.Find(m.resource.Id) {
		SendResourceContentToObserver(obs.client, m.contentCtx, obs.Observe(), obs.deviceID, obs.resourceID, obs.token)
	}
}

func (m *resourceCtx) TriggerSignIn(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.isPublished {
		m.onResourcePublishedLocked()
		m.onUpdateResourceLocked(ctx)
	}
}

func (m *resourceCtx) onUpdateResourceLocked(ctx context.Context) {
	client := m.server.clientContainerByDeviceID.Find(m.resource.DeviceId)
	if client == nil {
		return
	}
	for {
		if len(m.resourceUpdatePendings) == 0 {
			return
		}
		updatePending := m.resourceUpdatePendings[0]
		err := client.updateContent(ctx, m.resource, &updatePending)
		if err != nil {
			log.Errorf("DeviceId: %v, ResourceId: %v: cannot perform update: %v", m.resource.DeviceId, m.resource.Id, err)
			return
		}
		m.resourceUpdatePendings = m.resourceUpdatePendings[1:]
	}
}

func (m *resourceCtx) onRetrieveResourceLocked(ctx context.Context) {
	client := m.server.clientContainerByDeviceID.Find(m.resource.DeviceId)
	if client == nil {
		return
	}
	for {
		if len(m.resourceRetrievePendings) == 0 {
			return
		}
		retrievePending := m.resourceRetrievePendings[0]
		err := client.retrieveContent(ctx, m.resource, &retrievePending)
		if err != nil {
			log.Errorf("DeviceId: %v, ResourceId: %v: cannot perform retrieve: %v", m.resource.DeviceId, m.resource.Id, err)
			return
		}
		m.resourceRetrievePendings = m.resourceRetrievePendings[1:]
	}
}

func (m *resourceCtx) onResourceUpdatedLocked(updateProcessed []raEvents.ResourceUpdated) {
	for _, up := range updateProcessed {
		notify := m.server.updateNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource updated event", m.resource.DeviceId, m.resource.Id)
			}
		}
	}
}

func (m *resourceCtx) onResourceRetrievedLocked(resourceRetrieved []raEvents.ResourceRetrieved) {
	for _, up := range resourceRetrieved {
		notify := m.server.retrieveNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource retrieved event", m.resource.DeviceId, m.resource.Id)
			}
		}
	}
}

func (m *resourceCtx) SnapshotEventType() string {
	s := &raEvents.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}
func (m *resourceCtx) Content() *pbRA.ResourceChanged {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.contentCtx
}

func (m *resourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler
	var onResourcePublished, onResourceUnpublished, onResourceChanged bool
	resourceUpdatePendings := make([]raEvents.ResourceUpdatePending, 0, 16)
	resourceUpdated := make([]raEvents.ResourceUpdated, 0, 16)
	resourceRetrievePendings := make([]raEvents.ResourceRetrievePending, 0, 16)
	resourceRetrieved := make([]raEvents.ResourceRetrieved, 0, 16)
	m.lock.Lock()
	defer m.lock.Unlock()
	var anyEventProcessed bool
	for iter.Next(ctx, &eu) {
		anyEventProcessed = true
		log.Debugf("coap-gateway.resourceCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupId, eu.AggregateId, eu.Version, eu.EventType)
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
			m.contentCtx = s.GetLatestResourceChange()
			m.resource = s.Resource
			m.isPublished = s.IsPublished
			onResourceChanged = true
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
		case http.ProtobufContentType(&pbRA.ResourceUpdatePending{}):
			var s raEvents.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			resourceUpdatePendings = append(resourceUpdatePendings, s)
		case http.ProtobufContentType(&pbRA.ResourceUpdated{}):
			var s raEvents.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := resourceUpdatePendings[:0]
			for _, cu := range resourceUpdatePendings {
				if cu.AuditContext.CorrelationId != s.AuditContext.CorrelationId {
					tmp = append(tmp, cu)
				}
			}
			resourceUpdatePendings = tmp
			resourceUpdated = append(resourceUpdated, s)
		case http.ProtobufContentType(&pbRA.ResourceRetrievePending{}):
			var s raEvents.ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			resourceRetrievePendings = append(resourceRetrievePendings, s)
		case http.ProtobufContentType(&pbRA.ResourceRetrieved{}):
			var s raEvents.ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := resourceRetrievePendings[:0]
			for _, cu := range resourceRetrievePendings {
				if cu.AuditContext.CorrelationId != s.AuditContext.CorrelationId {
					tmp = append(tmp, cu)
				}
			}
			resourceRetrievePendings = tmp
			resourceRetrieved = append(resourceRetrieved, s)
		case http.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.contentCtx = &s.ResourceChanged
			onResourceChanged = true
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if m.resource == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", eu.GroupId, eu.AggregateId)
	}

	if onResourcePublished {
		m.onResourcePublishedLocked()
	} else if onResourceUnpublished {
		m.onResourceUnpublishedLocked()
	}

	if onResourceChanged && m.isPublished {
		m.onResourceChangedLocked()
	}

	m.onResourceUpdatedLocked(resourceUpdated)
	m.onResourceRetrievedLocked(resourceRetrieved)

	m.resourceUpdatePendings = append(m.resourceUpdatePendings, resourceUpdatePendings...)
	m.resourceRetrievePendings = append(m.resourceRetrievePendings, resourceRetrievePendings...)
	ctx, err := m.server.ctxWithServiceToken(ctx)
	if err != nil {
		log.Errorf("cannot update/restrieve resource /%v/%v: %v", m.resource.GetDeviceId(), m.resource.GetHref(), err)
		return nil
	}
	m.onUpdateResourceLocked(ctx)
	m.onRetrieveResourceLocked(ctx)

	return nil
}
