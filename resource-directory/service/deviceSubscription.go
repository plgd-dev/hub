package service

import (
	"context"
	"fmt"
	"hash/crc64"
	"sort"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type deviceSubscription struct {
	*subscription
	deviceEvent *pb.SubscribeToEvents_DeviceEventFilter

	isInitializedResourcePublishLock sync.Mutex
	isInitializedResourcePublish     bool
}

func NewDeviceSubscription(id, userID, token string, send SendEventFunc, resourceProjection *Projection, deviceEvent *pb.SubscribeToEvents_DeviceEventFilter) *deviceSubscription {
	log.Debugf("subscription.NewDeviceSubscription %v", id)
	defer log.Debugf("subscription.NewDeviceSubscription %v done", id)
	return &deviceSubscription{
		subscription: NewSubscription(userID, id, token, send, resourceProjection),
		deviceEvent:  deviceEvent,
	}
}

func (s *deviceSubscription) DeviceID() string {
	return s.deviceEvent.GetDeviceId()
}

type ResourceLinks struct {
	links   []*pb.ResourceLink
	version uint64
	isInit  bool
}

type SortResourceLinks []*pb.ResourceLink

func (a SortResourceLinks) Len() int {
	return len(a)
}

func (a SortResourceLinks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortResourceLinks) Less(i, j int) bool {
	if a[i].DeviceId < a[j].DeviceId {
		return true
	}
	if a[i].Href < a[j].Href {
		return true
	}
	return false
}

func CalcHashFromResourceLinks(action string, links []*pb.ResourceLink) uint64 {
	hash := crc64.New(crc64.MakeTable(crc64.ISO))
	hash.Write([]byte(action))
	for i := range links {
		hash.Write([]byte(links[i].DeviceId))
		hash.Write([]byte(links[i].Href))
		hash.Write([]byte(links[i].Title))
		for j := range links[i].Types {
			hash.Write([]byte(links[i].Types[j]))
		}
	}
	return hash.Sum64()
}

func (s *deviceSubscription) initializeResourcePublish(isInit bool) bool {
	s.isInitializedResourcePublishLock.Lock()
	defer s.isInitializedResourcePublishLock.Unlock()
	if isInit {
		s.isInitializedResourcePublish = true
	}
	return s.isInitializedResourcePublish
}

func (s *deviceSubscription) NotifyOfPublishedResourceLinks(ctx context.Context, links ResourceLinks) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_PUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if !s.initializeResourcePublish(links.isInit) {
		return nil
	}
	if len(links.links) == 0 {
		return nil
	}
	sort.Sort(SortResourceLinks(links.links))
	if s.FilterByVersionAndHash(links.links[0].GetDeviceId(), commands.ResourceLinksHref, "res", links.version, CalcHashFromResourceLinks("publish", links.links)) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourcePublished_{
			ResourcePublished: &pb.Event_ResourcePublished{
				Links: links.links,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUnpublishedResourceLinks(ctx context.Context, links ResourceLinks) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if len(links.links) == 0 {
		return nil
	}
	sort.Sort(SortResourceLinks(links.links))
	if s.FilterByVersionAndHash(links.links[0].GetDeviceId(), commands.ResourceLinksHref, "res", links.version, CalcHashFromResourceLinks("unpublish", links.links)) {
		return nil
	}

	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUnpublished_{
			ResourceUnpublished: &pb.Event_ResourceUnpublished{
				Links: links.links,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatePendingResource(ctx context.Context, updatePending *events.ResourceUpdatePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(updatePending.GetResourceId().GetDeviceId(), updatePending.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: updatePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatedResource(ctx context.Context, updated *events.ResourceUpdated, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UPDATED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdated{
			ResourceUpdated: updated,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievePendingResource(ctx context.Context, retrievePending *events.ResourceRetrievePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(retrievePending.GetResourceId().GetDeviceId(), retrievePending.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: retrievePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievedResource(ctx context.Context, retrieved *events.ResourceRetrieved, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_RETRIEVED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(retrieved.GetResourceId().GetDeviceId(), retrieved.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: retrieved,
		},
	})
}

func (s *deviceSubscription) NotifyOfDeletePendingResource(ctx context.Context, deletePending *events.ResourceDeletePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_DELETE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(deletePending.GetResourceId().GetDeviceId(), deletePending.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeletePending{
			ResourceDeletePending: deletePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfDeletedResource(ctx context.Context, deleted *events.ResourceDeleted, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_DELETED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(deleted.GetResourceId().GetDeviceId(), deleted.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeleted{
			ResourceDeleted: deleted,
		},
	})
}

func (s *deviceSubscription) NotifyOfCreatePendingResource(ctx context.Context, createPending *events.ResourceCreatePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_CREATE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(createPending.GetResourceId().GetDeviceId(), createPending.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: createPending,
		},
	})
}

func (s *deviceSubscription) NotifyOfCreatedResource(ctx context.Context, created *events.ResourceCreated, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_CREATED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersionAndHash(created.GetResourceId().GetDeviceId(), created.GetResourceId().GetHref(), "res", version, 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: created,
		},
	})
}

func (s *deviceSubscription) initGetResourceLinkProjection() eventstore.Model {
	s.isInitializedResourcePublishLock.Lock()
	defer s.isInitializedResourcePublishLock.Unlock()
	models := s.resourceProjection.Models(commands.NewResourceID(s.DeviceID(), commands.ResourceLinksHref))
	if len(models) != 1 {
		s.isInitializedResourcePublish = true
		return nil
	}
	return models[0]
}

func (s *deviceSubscription) initSendResourcesPublished(ctx context.Context) error {
	model := s.initGetResourceLinkProjection()
	if model == nil {
		return nil
	}

	rlp, ok := model.(*resourceLinksProjection)
	if !ok {
		return fmt.Errorf("unexpected event type")
	}

	err := rlp.InitialNotifyOfPublishedResourceLinks(ctx, s)
	if err != nil {
		return fmt.Errorf("cannot send resource published: %w", err)
	}

	return nil
}

func (s *deviceSubscription) initSendResourcesUnpublished(ctx context.Context) error {
	err := s.NotifyOfUnpublishedResourceLinks(ctx, ResourceLinks{})
	if err != nil {
		return fmt.Errorf("cannot send resource published: %w", err)
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesUpdatePending(ctx context.Context) error {
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[s.DeviceID()] {
		err := resource.OnResourceUpdatePendingLocked(ctx, s.NotifyOfUpdatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesRetrievePending(ctx context.Context) error {
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[s.DeviceID()] {
		err := resource.OnResourceRetrievePendingLocked(ctx, s.NotifyOfRetrievePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource retrieve pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesDeletePending(ctx context.Context) error {
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[s.DeviceID()] {
		err := resource.OnResourceDeletePendingLocked(ctx, s.NotifyOfDeletePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource delete pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesCreatePending(ctx context.Context) error {
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[s.DeviceID()] {
		err := resource.OnResourceCreatePendingLocked(ctx, s.NotifyOfCreatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	if !currentDevices[s.DeviceID()] {
		return fmt.Errorf("device %v not found", s.DeviceID())
	}
	_, err := s.RegisterToProjection(ctx, s.DeviceID())
	if err != nil {
		return fmt.Errorf("cannot register to resource projection: %w", err)
	}

	for _, f := range s.deviceEvent.GetEventsFilter() {
		switch f {
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_PUBLISHED:
			err = s.initSendResourcesPublished(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED:
			err = s.initSendResourcesUnpublished(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING:
			err = s.initSendResourcesUpdatePending(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING:
			err = s.initSendResourcesRetrievePending(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_DELETE_PENDING:
			err = s.initSendResourcesDeletePending(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_CREATE_PENDING:
			err = s.initSendResourcesCreatePending(ctx)
		case pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_UPDATED, pb.SubscribeToEvents_DeviceEventFilter_RESOURCE_RETRIEVED:
			// do nothing
		}
		if err != nil {
			return err
		}
	}
	return nil
}
