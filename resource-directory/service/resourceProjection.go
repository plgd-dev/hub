package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/log"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
)

type resourceProjection struct {
	lock                         sync.Mutex
	resourceId                   *commands.ResourceId
	content                      *events.ResourceChanged
	version                      uint64
	onResourcePublishedVersion   uint64
	onResourceUnpublishedVersion uint64
	onResourceChangedVersion     uint64

	subscriptions                 *Subscriptions
	updateNotificationContainer   *notification.UpdateNotificationContainer
	retrieveNotificationContainer *notification.RetrieveNotificationContainer
	deleteNotificationContainer   *notification.DeleteNotificationContainer
	resourceUpdatePendings        []events.ResourceUpdatePending
	resourceRetrievePendings      []events.ResourceRetrievePending
	resourceDeletePendings        []events.ResourceDeletePending
}

func NewResourceProjection(subscriptions *Subscriptions, updateNotificationContainer *notification.UpdateNotificationContainer, retrieveNotificationContainer *notification.RetrieveNotificationContainer, deleteNotificationContainer *notification.DeleteNotificationContainer) eventstore.Model {
	return &resourceProjection{
		subscriptions:                 subscriptions,
		updateNotificationContainer:   updateNotificationContainer,
		retrieveNotificationContainer: retrieveNotificationContainer,
		deleteNotificationContainer:   deleteNotificationContainer,
		resourceUpdatePendings:        make([]events.ResourceUpdatePending, 0, 8),
	}
}

func (rp *resourceProjection) cloneLocked() *resourceProjection {
	resourceUpdatePendings := make([]events.ResourceUpdatePending, 0, len(rp.resourceUpdatePendings))
	for _, v := range rp.resourceUpdatePendings {
		resourceUpdatePendings = append(resourceUpdatePendings, v)
	}
	return &resourceProjection{
		resourceId:             rp.resourceId,
		content:                rp.content,
		version:                rp.version,
		resourceUpdatePendings: resourceUpdatePendings,
	}
}

func (rp *resourceProjection) Clone() *resourceProjection {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	return rp.cloneLocked()
}

func (rp *resourceProjection) onResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error) error {
	if len(rp.resourceUpdatePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatePendingLocked /%v", rp.resourceId)
	for idx := range rp.resourceUpdatePendings {
		p := rp.resourceUpdatePendings[idx]
		updatePending := pb.Event_ResourceUpdatePending{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
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

func (rp *resourceProjection) sendEventResourceUpdated(ctx context.Context, resourcesUpdated []events.ResourceUpdated) error {
	for _, u := range resourcesUpdated {
		updated := pb.Event_ResourceUpdated{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := rp.subscriptions.OnResourceUpdated(ctx, updated, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (rp *resourceProjection) onResourceRetrievePendingLocked(ctx context.Context, do func(ctx context.Context, retrievePending pb.Event_ResourceRetrievePending, version uint64) error) error {
	if len(rp.resourceRetrievePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceRetrievePendingLocked /%v", rp.resourceId)
	for idx := range rp.resourceRetrievePendings {
		p := rp.resourceRetrievePendings[idx]
		retrievePending := pb.Event_ResourceRetrievePending{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
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

func (rp *resourceProjection) onResourceDeletePendingLocked(ctx context.Context, do func(ctx context.Context, deletePending pb.Event_ResourceDeletePending, version uint64) error) error {
	if len(rp.resourceDeletePendings) == 0 {
		return nil
	}
	log.Debugf("onResourceDeletePendingLocked /%v", rp.resourceId)
	for idx := range rp.resourceDeletePendings {
		p := rp.resourceDeletePendings[idx]
		deletePending := pb.Event_ResourceDeletePending{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
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

func (rp *resourceProjection) sendEventResourceRetrieved(ctx context.Context, resourcesRetrieved []events.ResourceRetrieved) error {
	for _, u := range resourcesRetrieved {
		retrieved := pb.Event_ResourceRetrieved{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := rp.subscriptions.OnResourceRetrieved(ctx, retrieved, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (rp *resourceProjection) sendEventResourceDeleted(ctx context.Context, resourceDeleted []events.ResourceDeleted) error {
	for _, u := range resourceDeleted {
		deleted := pb.Event_ResourceDeleted{
			ResourceId: &commands.ResourceId{
				DeviceId: rp.resourceId.GetDeviceId(),
				Href:     rp.resourceId.GetHref(),
			},
			Content:       pb.RAContent2Content(u.GetContent()),
			CorrelationId: u.GetAuditContext().GetCorrelationId(),
			Status:        pb.RAStatus2Status(u.GetStatus()),
		}
		err := rp.subscriptions.OnResourceDeleted(ctx, deleted, u.GetEventMetadata().GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func (rp *resourceProjection) onResourceChangedLocked(ctx context.Context, do func(ctx context.Context, resourceChanged pb.Event_ResourceChanged, version uint64) error) error {
	log.Debugf("onResourceChangedLocked %v %v", rp.resourceId, rp.onResourceChangedVersion)
	return do(ctx, pb.Event_ResourceChanged{
		ResourceId: &commands.ResourceId{
			DeviceId: rp.resourceId.GetDeviceId(),
			Href:     rp.resourceId.GetHref(),
		},
		Content: pb.RAContent2Content(rp.content.GetContent()),
		Status:  pb.RAStatus2Status(rp.content.GetStatus()),
	}, rp.onResourceChangedVersion)
}

func (rp *resourceProjection) onCloudStatusChangedLocked(ctx context.Context) error {
	log.Debugf("onCloudStatusChangedLocked %v", rp.resourceId)
	online, err := isDeviceOnline(rp.content.GetContent())
	if err != nil {
		return err
	}
	if online {
		return rp.subscriptions.OnDeviceOnline(ctx, DeviceIDVersion{
			deviceID: rp.resourceId.GetDeviceId(),
			version:  rp.onResourceChangedVersion,
		})
	}
	return rp.subscriptions.OnDeviceOffline(ctx, DeviceIDVersion{
		deviceID: rp.resourceId.GetDeviceId(),
		version:  rp.onResourceChangedVersion,
	})
}

func (rp *resourceProjection) onResourceUpdatedLocked(ctx context.Context, updateProcessed []events.ResourceUpdated) error {
	if len(updateProcessed) == 0 {
		return nil
	}
	log.Debugf("onResourceUpdatedLocked %v", rp.resourceId)
	for _, up := range updateProcessed {
		notify := rp.updateNotificationContainer.Find(up.GetAuditContext().GetCorrelationId())
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource updated event for %v", rp.resourceId)
			}
		}
	}
	return rp.sendEventResourceUpdated(ctx, updateProcessed)
}

func (rp *resourceProjection) onResourceRetrievedLocked(ctx context.Context, resourceRetrieved []events.ResourceRetrieved) error {
	if len(resourceRetrieved) == 0 {
		return nil
	}
	log.Debugf("onResourceRetrievedLocked %v", rp.resourceId)
	for _, up := range resourceRetrieved {
		notify := rp.retrieveNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource retrieved event for %v", rp.resourceId)
			}
		}
	}
	return rp.sendEventResourceRetrieved(ctx, resourceRetrieved)
}

func (rp *resourceProjection) onResourceDeletedLocked(ctx context.Context, resourceDeleted []events.ResourceDeleted) error {
	if len(resourceDeleted) == 0 {
		return nil
	}
	log.Debugf("onResourceDeletedLocked %v", rp.resourceId)
	for _, up := range resourceDeleted {
		notify := rp.deleteNotificationContainer.Find(up.AuditContext.CorrelationId)
		if notify != nil {
			select {
			case notify <- up:
			default:
				log.Debugf("cannot send resource deleted event for %v", rp.resourceId)
			}
		}
	}
	return rp.sendEventResourceDeleted(ctx, resourceDeleted)
}

func (rp *resourceProjection) SnapshotEventType() string {
	s := &events.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (rp *resourceProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onResourceContentChanged, onResourceUpdatePending, onResourceRetrievePending, onResourceDeletePending bool
	resourceUpdated := make([]events.ResourceUpdated, 0, 16)
	resourceRetrieved := make([]events.ResourceRetrieved, 0, 16)
	resourceDeleted := make([]events.ResourceDeleted, 0, 4)
	rp.lock.Lock()
	defer rp.lock.Unlock()
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
		rp.version = eu.Version()
		switch eu.EventType() {
		case (&events.ResourceStateSnapshotTaken{}).EventType():
			var s events.ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			rp.content = s.LatestResourceChange
			rp.onResourcePublishedVersion = eu.Version()
			rp.onResourceUnpublishedVersion = eu.Version()
			rp.onResourceChangedVersion = eu.Version()
			onResourceContentChanged = true
		case (&events.ResourceChanged{}).EventType():
			var s events.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			rp.content = &s
			rp.onResourceChangedVersion = eu.Version()
			onResourceContentChanged = true
		case (&events.ResourceUpdatePending{}).EventType():
			var s events.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceUpdatePendings = append(rp.resourceUpdatePendings, s)
			rp.resourceId = s.ResourceId
			onResourceUpdatePending = true
		case (&events.ResourceUpdated{}).EventType():
			var s events.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			tmp := make([]events.ResourceUpdatePending, 0, 16)
			var found bool
			for _, cu := range rp.resourceUpdatePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true
				}
			}
			if found {
				resourceUpdated = append(resourceUpdated, s)
				onResourceUpdatePending = true
				rp.resourceUpdatePendings = tmp
			}
		case (&events.ResourceRetrievePending{}).EventType():
			var s events.ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			rp.resourceRetrievePendings = append(rp.resourceRetrievePendings, s)
			onResourceRetrievePending = true
		case (&events.ResourceDeletePending{}).EventType():
			var s events.ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			rp.resourceDeletePendings = append(rp.resourceDeletePendings, s)
			onResourceDeletePending = true
		case (&events.ResourceRetrieved{}).EventType():
			var s events.ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			tmp := make([]events.ResourceRetrievePending, 0, 16)
			var found bool
			for _, cu := range rp.resourceRetrievePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true

				}
			}
			if found {
				resourceRetrieved = append(resourceRetrieved, s)
				onResourceRetrievePending = true
				rp.resourceRetrievePendings = tmp
			}
		case (&events.ResourceDeleted{}).EventType():
			var s events.ResourceDeleted
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			rp.resourceId = s.ResourceId
			tmp := make([]events.ResourceDeletePending, 0, 16)
			var found bool
			for _, cu := range rp.resourceDeletePendings {
				if cu.GetAuditContext().GetCorrelationId() != s.GetAuditContext().GetCorrelationId() {
					tmp = append(tmp, cu)
				} else {
					found = true

				}
			}
			if found {
				resourceDeleted = append(resourceDeleted, s)
				onResourceDeletePending = true
				rp.resourceDeletePendings = tmp
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if rp.resourceId == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", groupID, aggregateID)
	}

	if onResourceContentChanged {
		if rp.resourceId.GetHref() == commands.StatusHref {
			if err := rp.onCloudStatusChangedLocked(ctx); err != nil {
				log.Errorf("cannot make action on cloud status changed: %v", err)
			}
		}

		if err := rp.onResourceChangedLocked(ctx, rp.subscriptions.OnResourceContentChanged); err != nil {
			log.Errorf("%v", err)
		}
	}

	if onResourceUpdatePending {
		err := rp.onResourceUpdatePendingLocked(ctx, rp.subscriptions.OnResourceUpdatePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if onResourceRetrievePending {
		err := rp.onResourceRetrievePendingLocked(ctx, rp.subscriptions.OnResourceRetrievePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if onResourceDeletePending {
		err := rp.onResourceDeletePendingLocked(ctx, rp.subscriptions.OnResourceDeletePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}

	err := rp.onResourceUpdatedLocked(ctx, resourceUpdated)
	if err != nil {
		log.Errorf("%v", err)
	}
	err = rp.onResourceRetrievedLocked(ctx, resourceRetrieved)
	if err != nil {
		log.Errorf("%v", err)
	}
	err = rp.onResourceDeletedLocked(ctx, resourceDeleted)
	if err != nil {
		log.Errorf("%v", err)
	}

	return nil
}
