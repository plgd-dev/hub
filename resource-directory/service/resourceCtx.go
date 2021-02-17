package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/log"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
)

type resourceCtx struct {
	lock                         sync.Mutex
	resourceId                   *commands.ResourceId
	content                      *events.ResourceChanged
	version                      uint64
	onResourcePublishedVersion   uint64
	onResourceUnpublishedVersion uint64
	onResourceChangedVersion     uint64

	subscriptions                 *subscriptions
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	resourceUpdatePendings        []events.ResourceUpdatePending
	resourceRetrievePendings      []events.ResourceRetrievePending
	resourceDeletePendings        []events.ResourceDeletePending
}

func NewResourceCtx(subscriptions *subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer, deleteNotificationContainer *notification.DeleteNotificationContainer) func(context.Context) eventstore.Model {
	return func(ctx context.Context) eventstore.Model {
		return &resourceCtx{
			subscriptions:                 subscriptions,
			updateNotificationContainer:   updateNotificationContainer,
			retrieveNotificationContainer: retrieveNotificationContainer,
			deleteNotificationContainer:   deleteNotificationContainer,
			resourceUpdatePendings:        make([]events.ResourceUpdatePending, 0, 8),
		}
	}
}

func (m *resourceCtx) cloneLocked() *resourceCtx {
	resourceUpdatePendings := make([]events.ResourceUpdatePending, 0, len(m.resourceUpdatePendings))
	for _, v := range m.resourceUpdatePendings {
		resourceUpdatePendings = append(resourceUpdatePendings, v)
	}
	return &resourceCtx{
		resourceId:             m.resourceId,
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

func (m *resourceCtx) onResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error) error {
	if len(m.resourceUpdatePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatePendingLocked /%v", m.resourceId)
	for idx := range m.resourceUpdatePendings {
		p := m.resourceUpdatePendings[idx]
		updatePending := pb.Event_ResourceUpdatePending{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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

func (m *resourceCtx) sendEventResourceUpdated(ctx context.Context, resourcesUpdated []events.ResourceUpdated) error {
	for _, u := range resourcesUpdated {
		updated := pb.Event_ResourceUpdated{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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
	log.Debugf("onResourceRetrievePendingLocked /%v", m.resourceId)
	for idx := range m.resourceRetrievePendings {
		p := m.resourceRetrievePendings[idx]
		retrievePending := pb.Event_ResourceRetrievePending{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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
	log.Debugf("onResourceDeletePendingLocked /%v", m.resourceId)
	for idx := range m.resourceDeletePendings {
		p := m.resourceDeletePendings[idx]
		deletePending := pb.Event_ResourceDeletePending{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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

func (m *resourceCtx) sendEventResourceRetrieved(ctx context.Context, resourcesRetrieved []events.ResourceRetrieved) error {
	for _, u := range resourcesRetrieved {
		retrieved := pb.Event_ResourceRetrieved{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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

func (m *resourceCtx) sendEventResourceDeleted(ctx context.Context, resourceDeleted []events.ResourceDeleted) error {
	for _, u := range resourceDeleted {
		deleted := pb.Event_ResourceDeleted{
			ResourceId: &commands.ResourceId{
				DeviceId: m.resourceId.GetDeviceId(),
				Href:     m.resourceId.GetHref(),
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
	log.Debugf("onResourceChangedLocked %v%v %v", m.resourceId, m.onResourceChangedVersion)
	return do(ctx, pb.Event_ResourceChanged{
		ResourceId: &commands.ResourceId{
			DeviceId: m.resourceId.GetDeviceId(),
			Href:     m.resourceId.GetHref(),
		},
		Content: pb.RAContent2Content(m.content.GetContent()),
		Status:  pb.RAStatus2Status(m.content.GetStatus()),
	}, m.onResourceChangedVersion)
}

func (m *resourceCtx) onCloudStatusChangedLocked(ctx context.Context) error {
	log.Debugf("onCloudStatusChangedLocked %v", m.resourceId)
	online, err := isDeviceOnline(m.content.GetContent())
	if err != nil {
		return err
	}
	if online {
		return m.subscriptions.OnDeviceOnline(ctx, DeviceIDVersion{
			deviceID: m.resourceId.GetDeviceId(),
			version:  m.onResourceChangedVersion,
		})
	}
	return m.subscriptions.OnDeviceOffline(ctx, DeviceIDVersion{
		deviceID: m.resourceId.GetDeviceId(),
		version:  m.onResourceChangedVersion,
	})
}

func (m *resourceCtx) onResourceUpdatedLocked(ctx context.Context, updateProcessed []events.ResourceUpdated) error {
	if len(updateProcessed) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatedLocked %v", m.resourceId)
	for _, up := range updateProcessed {
		notify := m.updateNotificationContainer.Find(up.GetAuditContext().GetCorrelationId())
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource updated event for %v", m.resourceId)
			}
		}
	}
	return m.sendEventResourceUpdated(ctx, updateProcessed)
}

func (m *resourceCtx) onResourceRetrievedLocked(ctx context.Context, resourceRetrieved []events.ResourceRetrieved) error {
	if len(resourceRetrieved) == 0 {
		return nil
	}
	log.Debugf("onResourceRetrievedLocked %v", m.resourceId)
	for _, up := range resourceRetrieved {
		notify := m.retrieveNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource retrieved event for %v", m.resourceId)
			}
		}
	}
	return m.sendEventResourceRetrieved(ctx, resourceRetrieved)
}

func (m *resourceCtx) onResourceDeletedLocked(ctx context.Context, resourceDeleted []events.ResourceDeleted) error {
	if len(resourceDeleted) == 0 {
		return nil
	}
	log.Debugf("onResourceDeletedLocked %v", m.resourceId)
	for _, up := range resourceDeleted {
		notify := m.deleteNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource deleted event for %v", m.resourceId)
			}
		}
	}
	return m.sendEventResourceDeleted(ctx, resourceDeleted)
}

func (m *resourceCtx) SnapshotEventType() string {
	s := &events.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *resourceCtx) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onResourceContentChanged, onResourceUpdatePending, onResourceRetrievePending, onResourceDeletePending bool
	resourceUpdated := make([]events.ResourceUpdated, 0, 16)
	resourceRetrieved := make([]events.ResourceRetrieved, 0, 16)
	resourceDeleted := make([]events.ResourceDeleted, 0, 4)
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
		case (&events.ResourceStateSnapshotTaken{}).EventType():
			var s events.ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.content = s.LatestResourceChange
			m.resourceId = s.ResourceId
			m.onResourcePublishedVersion = eu.Version()
			m.onResourceUnpublishedVersion = eu.Version()
			m.onResourceChangedVersion = eu.Version()
			onResourceContentChanged = true
		case (&events.ResourceChanged{}).EventType():
			var s events.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.content = &s
			m.onResourceChangedVersion = eu.Version()
			onResourceContentChanged = true
		case (&events.ResourceUpdatePending{}).EventType():
			var s events.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceUpdatePendings = append(m.resourceUpdatePendings, s)
			onResourceUpdatePending = true
		case (&events.ResourceUpdated{}).EventType():
			var s events.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]events.ResourceUpdatePending, 0, 16)
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
		case (&events.ResourceRetrievePending{}).EventType():
			var s events.ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceRetrievePendings = append(m.resourceRetrievePendings, s)
			onResourceRetrievePending = true
		case (&events.ResourceDeletePending{}).EventType():
			var s events.ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.resourceDeletePendings = append(m.resourceDeletePendings, s)
			onResourceDeletePending = true
		case (&events.ResourceRetrieved{}).EventType():
			var s events.ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]events.ResourceRetrievePending, 0, 16)
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
		case (&events.ResourceDeleted{}).EventType():
			var s events.ResourceDeleted
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]events.ResourceDeletePending, 0, 16)
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

	if m.resourceId == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
	}

	if onResourceContentChanged {
		if m.resourceId.GetHref() == status.Href {
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
