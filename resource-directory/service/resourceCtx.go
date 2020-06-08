package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/sdk/schema/cloud"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

type resourceCtx struct {
	lock                         sync.Mutex
	resource                     *pbRA.Resource
	isPublished                  bool
	content                      *pbRA.ResourceChanged
	version                      uint64
	onResourcePublishedVersion   uint64
	onResourceUnpublishedVersion uint64
	onResourceChangedVersion     uint64

	subscriptions                 *subscriptions
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	resourceUpdatePendings        []raEvents.ResourceUpdatePending
}

func NewResourceCtx(subscriptions *subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer) func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			subscriptions:                 subscriptions,
			updateNotificationContainer:   updateNotificationContainer,
			retrieveNotificationContainer: retrieveNotificationContainer,
			resourceUpdatePendings:        make([]raEvents.ResourceUpdatePending, 0, 8),
		}, nil
	}
}

func (m *resourceCtx) cloneLocked() *resourceCtx {
	resourceUpdatePendings := make([]raEvents.ResourceUpdatePending, 0, len(m.resourceUpdatePendings))
	for _, v := range m.resourceUpdatePendings {
		resourceUpdatePendings = append(resourceUpdatePendings, v)
	}
	return &resourceCtx{
		resource:               m.resource,
		isPublished:            m.isPublished,
		content:                m.content,
		version:                m.version,
		resourceUpdatePendings: resourceUpdatePendings,
	}
}

func (m *resourceCtx) Clone() *resourceCtx {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.cloneLocked()
}

func (m *resourceCtx) onResourcePublishedLocked(ctx context.Context) error {
	log.Debugf("onResourcePublishedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	link := pb.RAResourceToProto(m.resource)
	return m.subscriptions.OnResourcePublished(ctx, link, m.onResourcePublishedVersion)
}

func (m *resourceCtx) onResourceUnpublishedLocked(ctx context.Context) error {
	log.Debugf("onResourceUnpublishedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	link := pb.RAResourceToProto(m.resource)
	return m.subscriptions.OnResourceUnpublished(ctx, link, m.onResourceUnpublishedVersion)
}

func (m *resourceCtx) onResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error) error {
	log.Debugf("onResourceUpdatePending %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for idx := range m.resourceUpdatePendings {
		p := m.resourceUpdatePendings[idx]
		updatePending := pb.Event_ResourceUpdatePending{
			ResourceId: &pb.ResourceId{
				DeviceId:         m.resource.GetDeviceId(),
				ResourceLinkHref: m.resource.GetHref(),
			},
			ResourceInterface: p.GetResourceInterface(),
			Content:           pb.RAContent2Content(p.GetContent()),
			CorrelationId:     p.GetAuditContext().GetCorrelationId(),
		}
		err := do(ctx, updatePending, p.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) sendEventResourceUpdated(ctx context.Context, resourcesUpdated []raEvents.ResourceUpdated) error {
	for _, u := range resourcesUpdated {
		updated := pb.Event_ResourceUpdated{
			ResourceId: &pb.ResourceId{
				DeviceId:         m.resource.GetDeviceId(),
				ResourceLinkHref: m.resource.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := m.subscriptions.OnResourceUpdated(ctx, updated, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) onResourceChangedLocked(ctx context.Context) error {
	log.Debugf("onResourceChangedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	if m.content.GetStatus() != pbRA.Status_OK {
		err := fmt.Errorf("unable to subscribe to resource %v%v: device response: %v", m.resource.GetDeviceId(), m.resource.GetHref(), m.content.GetStatus())
		m.subscriptions.CancelResourceSubscriptions(ctx, m.resource.GetDeviceId(), m.resource.GetHref(), err)
		return err
	}
	content := makeContent(m.content.GetContent())
	return m.subscriptions.OnResourceContentChanged(ctx, m.resource.GetDeviceId(), m.resource.GetHref(), content, m.onResourceChangedVersion)
}

func (m *resourceCtx) onCloudStatusChangedLocked(ctx context.Context) error {
	log.Debugf("onCloudStatusChangedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	online, err := isDeviceOnline(m.content.GetContent())
	if err != nil {
		return err
	}
	if online {
		return m.subscriptions.OnDeviceOnline(ctx, m.resource.GetDeviceId(), m.onResourceChangedVersion)
	}
	return m.subscriptions.OnDeviceOffline(ctx, m.resource.GetDeviceId(), m.onResourceChangedVersion)
}

func (m *resourceCtx) onResourceUpdatedLocked(ctx context.Context, updateProcessed []raEvents.ResourceUpdated) error {
	log.Debugf("onResourceUpdatedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for _, up := range updateProcessed {
		notify := m.updateNotificationContainer.Find(up.GetAuditContext().GetCorrelationId())
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource updated event", m.resource.DeviceId, m.resource.Id)
			}
		}
	}
	return m.sendEventResourceUpdated(ctx, updateProcessed)
}

func (m *resourceCtx) onResourceRetrievedLocked(resourceRetrieved []raEvents.ResourceRetrieved) {
	log.Debugf("onResourceRetrievedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for _, up := range resourceRetrieved {
		notify := m.retrieveNotificationContainer.Find(up.AuditContext.CorrelationId)
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

func (m *resourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler
	var onResourcePublished, onResourceUnpublished, onResourceContentChanged bool
	resourceUpdated := make([]raEvents.ResourceUpdated, 0, 16)
	resourceRetrieved := make([]raEvents.ResourceRetrieved, 0, 16)
	resourceUpdatePendings := make([]raEvents.ResourceUpdatePending, 0, 16)
	m.lock.Lock()
	defer m.lock.Unlock()
	var anyEventProcessed bool
	for iter.Next(ctx, &eu) {
		anyEventProcessed = true
		log.Debugf("grpc-gateway.resourceCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupId, eu.AggregateId, eu.Version, eu.EventType)
		m.version = eu.Version
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
			m.content = s.LatestResourceChange
			m.resource = s.Resource
			m.isPublished = s.IsPublished
			m.onResourcePublishedVersion = eu.Version
			m.onResourceUnpublishedVersion = eu.Version
			m.onResourceChangedVersion = eu.Version
			onResourceContentChanged = true
		case http.ProtobufContentType(&pbRA.ResourcePublished{}):
			var s raEvents.ResourcePublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if !m.isPublished {
				onResourcePublished = true
				onResourceUnpublished = false
			}
			m.onResourcePublishedVersion = eu.Version
			m.isPublished = true
			m.resource = s.Resource
		case http.ProtobufContentType(&pbRA.ResourceUnpublished{}):
			if m.isPublished {
				onResourcePublished = false
				onResourceUnpublished = true
			}
			m.onResourceUnpublishedVersion = eu.Version
			m.isPublished = false
		case http.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.content = &s.ResourceChanged
			m.onResourceChangedVersion = eu.Version
			onResourceContentChanged = true
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
			tmp := make([]raEvents.ResourceUpdatePending, 0, 16)
			var found bool
			for _, cu := range resourceUpdatePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true
				}
			}
			if !found {
				for _, cu := range m.resourceUpdatePendings {
					if cu.GetAuditContext().GetCorrelationId() == s.GetAuditContext().GetCorrelationId() {
						found = true
					}
				}
			}
			if found {
				resourceUpdated = append(resourceUpdated, s)
			}
			resourceUpdatePendings = tmp
		case http.ProtobufContentType(&pbRA.ResourceRetrieved{}):
			var s raEvents.ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			resourceRetrieved = append(resourceRetrieved, s)
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
		if err := m.onResourcePublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	} else if onResourceUnpublished {
		if err := m.onResourceUnpublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	if onResourceContentChanged && m.isPublished {
		if cqrsRA.MakeResourceId(m.resource.GetDeviceId(), cloud.StatusHref) == m.resource.Id {
			if err := m.onCloudStatusChangedLocked(ctx); err != nil {
				log.Errorf("cannot make action on cloud status changed: %v", err)
			}
		}

		if err := m.onResourceChangedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	m.resourceUpdatePendings = append(m.resourceUpdatePendings, resourceUpdatePendings...)
	err := m.onResourceUpdatePendingLocked(ctx, m.subscriptions.OnResourceUpdatePending)
	if err != nil {
		log.Errorf("%v", err)
	}

	err = m.onResourceUpdatedLocked(ctx, resourceUpdated)
	if err != nil {
		log.Errorf("%v", err)
	}
	m.onResourceRetrievedLocked(resourceRetrieved)

	return nil
}
