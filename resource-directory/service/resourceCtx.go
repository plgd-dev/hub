package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/sdk/schema/cloud"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
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
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	resourceUpdatePendings        []raEvents.ResourceUpdatePending
	resourceRetrievePendings      []raEvents.ResourceRetrievePending
	resourceDeletePendings        []raEvents.ResourceDeletePending
}

func NewResourceCtx(subscriptions *subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer, deleteNotificationContainer *notification.DeleteNotificationContainer) func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			subscriptions:                 subscriptions,
			updateNotificationContainer:   updateNotificationContainer,
			retrieveNotificationContainer: retrieveNotificationContainer,
			deleteNotificationContainer:   deleteNotificationContainer,
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
	return m.subscriptions.OnResourcePublished(ctx, ResourceLink{
		link:    link,
		version: m.onResourcePublishedVersion,
	})
}

func (m *resourceCtx) onResourceUnpublishedLocked(ctx context.Context) error {
	log.Debugf("onResourceUnpublishedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	link := pb.RAResourceToProto(m.resource)
	return m.subscriptions.OnResourceUnpublished(ctx, ResourceLink{
		link:    link,
		version: m.onResourceUnpublishedVersion,
	})
}

func (m *resourceCtx) onResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error) error {
	if len(m.resourceUpdatePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatePendingLocked /%v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for idx := range m.resourceUpdatePendings {
		p := m.resourceUpdatePendings[idx]
		updatePending := pb.Event_ResourceUpdatePending{
			ResourceId: &pb.ResourceId{
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
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
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
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

func (m *resourceCtx) onResourceRetrievePendingLocked(ctx context.Context, do func(ctx context.Context, retrievePending pb.Event_ResourceRetrievePending, version uint64) error) error {
	if len(m.resourceRetrievePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceRetrievePendingLocked /%v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for idx := range m.resourceRetrievePendings {
		p := m.resourceRetrievePendings[idx]
		retrievePending := pb.Event_ResourceRetrievePending{
			ResourceId: &pb.ResourceId{
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
			},
			ResourceInterface: p.GetResourceInterface(),
			CorrelationId:     p.GetAuditContext().GetCorrelationId(),
		}
		err := do(ctx, retrievePending, p.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) onResourceDeletePendingLocked(ctx context.Context, do func(ctx context.Context, deletePending pb.Event_ResourceDeletePending, version uint64) error) error {
	if len(m.resourceDeletePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceDeletePendingLocked /%v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for idx := range m.resourceDeletePendings {
		p := m.resourceDeletePendings[idx]
		deletePending := pb.Event_ResourceDeletePending{
			ResourceId: &pb.ResourceId{
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
			},
			CorrelationId: p.GetAuditContext().GetCorrelationId(),
		}
		err := do(ctx, deletePending, p.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) sendEventResourceRetrieved(ctx context.Context, resourcesRetrieved []raEvents.ResourceRetrieved) error {
	for _, u := range resourcesRetrieved {
		retrieved := pb.Event_ResourceRetrieved{
			ResourceId: &pb.ResourceId{
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := m.subscriptions.OnResourceRetrieved(ctx, retrieved, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) sendEventResourceDeleted(ctx context.Context, resourceDeleted []raEvents.ResourceDeleted) error {
	for _, u := range resourceDeleted {
		deleted := pb.Event_ResourceDeleted{
			ResourceId: &pb.ResourceId{
				DeviceId: m.resource.GetDeviceId(),
				Href:     m.resource.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := m.subscriptions.OnResourceDeleted(ctx, deleted, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *resourceCtx) onResourceChangedLocked(ctx context.Context, do func(ctx context.Context, resourceChanged pb.Event_ResourceChanged, version uint64) error) error {
	log.Debugf("onResourceChangedLocked %v%v %v", m.resource.GetDeviceId(), m.resource.GetHref(), m.onResourceChangedVersion)
	return do(ctx, pb.Event_ResourceChanged{
		ResourceId: &pb.ResourceId{
			DeviceId: m.resource.GetDeviceId(),
			Href:     m.resource.GetHref(),
		},
		Content: pb.RAContent2Content(m.content.GetContent()),
		Status:  pb.RAStatus2Status(m.content.GetStatus()),
	}, m.onResourceChangedVersion)
}

func (m *resourceCtx) onCloudStatusChangedLocked(ctx context.Context) error {
	log.Debugf("onCloudStatusChangedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	online, err := isDeviceOnline(m.content.GetContent())
	if err != nil {
		return err
	}
	if online {
		return m.subscriptions.OnDeviceOnline(ctx, DeviceIDVersion{
			deviceID: m.resource.GetDeviceId(),
			version:  m.onResourceChangedVersion,
		})
	}
	return m.subscriptions.OnDeviceOffline(ctx, DeviceIDVersion{
		deviceID: m.resource.GetDeviceId(),
		version:  m.onResourceChangedVersion,
	})
}

func (m *resourceCtx) onResourceUpdatedLocked(ctx context.Context, updateProcessed []raEvents.ResourceUpdated) error {
	if len(updateProcessed) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for _, up := range updateProcessed {
		notify := m.updateNotificationContainer.Find(up.GetAuditContext().GetCorrelationId())
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource updated event", m.resource.GetDeviceId(), m.resource.GetId())
			}
		}
	}
	return m.sendEventResourceUpdated(ctx, updateProcessed)
}

func (m *resourceCtx) onResourceRetrievedLocked(ctx context.Context, resourceRetrieved []raEvents.ResourceRetrieved) error {
	if len(resourceRetrieved) == 0 {
		return nil
	}
	log.Debugf("onResourceRetrievedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for _, up := range resourceRetrieved {
		notify := m.retrieveNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource retrieved event", m.resource.GetDeviceId(), m.resource.GetId())
			}
		}
	}
	return m.sendEventResourceRetrieved(ctx, resourceRetrieved)
}

func (m *resourceCtx) onResourceDeletedLocked(ctx context.Context, resourceDeleted []raEvents.ResourceDeleted) error {
	if len(resourceDeleted) == 0 {
		return nil
	}
	log.Debugf("onResourceDeletedLocked %v%v", m.resource.GetDeviceId(), m.resource.GetHref())
	for _, up := range resourceDeleted {
		notify := m.deleteNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("DeviceId: %v, ResourceId: %v: cannot send resource deleted event", m.resource.GetDeviceId(), m.resource.GetId())
			}
		}
	}
	return m.sendEventResourceDeleted(ctx, resourceDeleted)
}

func (m *resourceCtx) SnapshotEventType() string {
	s := &raEvents.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *resourceCtx) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onResourcePublished, onResourceUnpublished, onResourceContentChanged, onResourceUpdatePending, onResourceRetrievePending, onResourceDeletePending bool
	resourceUpdated := make([]raEvents.ResourceUpdated, 0, 16)
	resourceRetrieved := make([]raEvents.ResourceRetrieved, 0, 16)
	resourceDeleted := make([]raEvents.ResourceDeleted, 0, 4)
	m.lock.Lock()
	defer m.lock.Unlock()
	var anyEventProcessed bool
	var groupID, aggregateID string
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		groupID = eu.GroupID()
		aggregateID = eu.AggregateID()
		anyEventProcessed = true
		log.Debugf("grpc-gateway.resourceCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupID(), eu.AggregateID(), eu.Version(), eu.EventType())
		m.version = eu.Version()
		switch eu.EventType() {
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
			m.onResourcePublishedVersion = eu.Version()
			m.onResourceUnpublishedVersion = eu.Version()
			m.onResourceChangedVersion = eu.Version()
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
			m.onResourcePublishedVersion = eu.Version()
			m.isPublished = true
			m.resource = s.Resource
		case http.ProtobufContentType(&pbRA.ResourceUnpublished{}):
			if m.isPublished {
				onResourcePublished = false
				onResourceUnpublished = true
			}
			m.onResourceUnpublishedVersion = eu.Version()
			m.isPublished = false
		case http.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.content = &s.ResourceChanged
			m.onResourceChangedVersion = eu.Version()
			onResourceContentChanged = true
		case http.ProtobufContentType(&pbRA.ResourceUpdatePending{}):
			var s raEvents.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceUpdatePendings = append(m.resourceUpdatePendings, s)
			onResourceUpdatePending = true
		case http.ProtobufContentType(&pbRA.ResourceUpdated{}):
			var s raEvents.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]raEvents.ResourceUpdatePending, 0, 16)
			var found bool
			for _, cu := range m.resourceUpdatePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true
				}
			}
			if found {
				resourceUpdated = append(resourceUpdated, s)
				onResourceUpdatePending = true
				m.resourceUpdatePendings = tmp
			}
		case http.ProtobufContentType(&pbRA.ResourceRetrievePending{}):
			var s raEvents.ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceRetrievePendings = append(m.resourceRetrievePendings, s)
			onResourceRetrievePending = true
		case http.ProtobufContentType(&pbRA.ResourceDeletePending{}):
			var s raEvents.ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceDeletePendings = append(m.resourceDeletePendings, s)
			onResourceDeletePending = true
		case http.ProtobufContentType(&pbRA.ResourceRetrieved{}):
			var s raEvents.ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]raEvents.ResourceRetrievePending, 0, 16)
			var found bool
			for _, cu := range m.resourceRetrievePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true

				}
			}
			if found {
				resourceRetrieved = append(resourceRetrieved, s)
				onResourceRetrievePending = true
				m.resourceRetrievePendings = tmp
			}
		case http.ProtobufContentType(&pbRA.ResourceDeleted{}):
			var s raEvents.ResourceDeleted
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]raEvents.ResourceDeletePending, 0, 16)
			var found bool
			for _, cu := range m.resourceDeletePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true

				}
			}
			if found {
				resourceDeleted = append(resourceDeleted, s)
				onResourceDeletePending = true
				m.resourceDeletePendings = tmp
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if m.resource == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
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
		if utils.MakeResourceId(m.resource.GetDeviceId(), cloud.StatusHref) == m.resource.GetId() {
			if err := m.onCloudStatusChangedLocked(ctx); err != nil {
				log.Errorf("cannot make action on cloud status changed: %v", err)
			}
		}

		if err := m.onResourceChangedLocked(ctx, m.subscriptions.OnResourceContentChanged); err != nil {
			log.Errorf("%v", err)
		}
	}

	if onResourceUpdatePending {
		err := m.onResourceUpdatePendingLocked(ctx, m.subscriptions.OnResourceUpdatePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if onResourceRetrievePending {
		err := m.onResourceRetrievePendingLocked(ctx, m.subscriptions.OnResourceRetrievePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if onResourceDeletePending {
		err := m.onResourceDeletePendingLocked(ctx, m.subscriptions.OnResourceDeletePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}

	err := m.onResourceUpdatedLocked(ctx, resourceUpdated)
	if err != nil {
		log.Errorf("%v", err)
	}
	err = m.onResourceRetrievedLocked(ctx, resourceRetrieved)
	if err != nil {
		log.Errorf("%v", err)
	}
	err = m.onResourceDeletedLocked(ctx, resourceDeleted)
	if err != nil {
		log.Errorf("%v", err)
	}

	return nil
}
