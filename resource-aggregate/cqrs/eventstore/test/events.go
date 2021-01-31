package test

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	httpUtils "github.com/plgd-dev/kit/net/http"
)

func MakeResourcePublishedEvent(resource pb.Resource, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rp := events.ResourceLinksPublished{
		ResourceLinksPublished: pb.ResourceLinksPublished{
			Id:       resource.Id,
			Resource: &resource,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: resource.DeviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		rp.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourcePublished{}),
		rp.Id,
		rp.Resource.DeviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourcePublished); ok {
				*x = rp
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceLinksUnpublishedEvent(id, deviceID string, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	ru := events.ResourceLinksUnpublished{
		ResourceLinksUnpublished: pb.ResourceLinksUnpublished{
			Id: id,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceID,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		ru.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceLinksUnpublished{}),
		ru.Id,
		deviceID,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksUnpublished); ok {
				*x = ru
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceStateSnapshotTaken(isPublished bool, resource pb.Resource, latestResourceChange pb.ResourceChanged, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rs := events.NewResourceStateSnapshotTaken()
	rs.Id = resource.Id
	rs.Resource = &resource
	rs.LatestResourceChange = &latestResourceChange
	rs.EventMetadata = &eventMetadata

	return eventstore.NewLoadedEvent(
		rs.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceStateSnapshotTaken{}),
		rs.Id,
		rs.Resource.DeviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceStateSnapshotTaken); ok {
				*x = *rs
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceUpdatePending(deviceId, resourceId string, content pb.Content, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rc := events.ResourceUpdatePending{
		ResourceUpdatePending: pb.ResourceUpdatePending{
			Id:      resourceId,
			Content: &content,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		rc.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceUpdatePending{}),
		rc.Id,
		deviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdatePending); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceUpdated(deviceId, resourceId string, status pb.Status, content pb.Content, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rc := events.ResourceUpdated{
		ResourceUpdated: pb.ResourceUpdated{
			Id:      resourceId,
			Content: &content,
			Status:  status,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		rc.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceUpdated{}),
		rc.Id,
		deviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdated); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceChangedEvent(id, deviceID string, content pb.Content, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	ru := events.ResourceChanged{
		ResourceChanged: pb.ResourceChanged{
			Id: id,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceID,
			},
			Content:       &content,
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		ru.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceChanged{}),
		ru.Id,
		deviceID,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceChanged); ok {
				*x = ru
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrievePending(deviceId, resourceId string, resourceInterface string, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rc := events.ResourceRetrievePending{
		ResourceRetrievePending: pb.ResourceRetrievePending{
			Id:                resourceId,
			ResourceInterface: resourceInterface,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		rc.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceRetrievePending{}),
		rc.Id,
		deviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrievePending); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrieved(deviceId, resourceId string, status pb.Status, content pb.Content, eventMetadata pb.EventMetadata) eventstore.EventUnmarshaler {
	rc := events.ResourceRetrieved{
		ResourceRetrieved: pb.ResourceRetrieved{
			Id:      resourceId,
			Content: &content,
			Status:  status,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return eventstore.NewLoadedEvent(
		rc.EventMetadata.Version,
		httpUtils.ProtobufContentType(&pb.ResourceRetrieved{}),
		rc.Id,
		deviceId,
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrieved); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}
