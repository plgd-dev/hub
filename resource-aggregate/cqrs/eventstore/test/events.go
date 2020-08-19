package test

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/cqrs/event"
	httpUtils "github.com/plgd-dev/kit/net/http"
)

func MakeResourcePublishedEvent(resource pb.Resource, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
	rp := events.ResourcePublished{
		ResourcePublished: pb.ResourcePublished{
			Id:       resource.Id,
			Resource: &resource,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: resource.DeviceId,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return event.EventUnmarshaler{
		Version:     rp.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourcePublished{}),
		AggregateId: rp.Id,
		GroupId:     rp.Resource.DeviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourcePublished); ok {
				*x = rp
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceUnpublishedEvent(id, deviceID string, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
	ru := events.ResourceUnpublished{
		ResourceUnpublished: pb.ResourceUnpublished{
			Id: id,
			AuditContext: &pb.AuditContext{
				UserId:   "userId",
				DeviceId: deviceID,
			},
			EventMetadata: &eventMetadata,
		},
	}
	return event.EventUnmarshaler{
		Version:     ru.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceUnpublished{}),
		AggregateId: ru.Id,
		GroupId:     deviceID,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceUnpublished); ok {
				*x = ru
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceStateSnapshotTaken(isPublished bool, resource pb.Resource, latestResourceChange pb.ResourceChanged, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
	rs := events.NewResourceStateSnapshotTaken(func(string, string) error { return nil })
	rs.Id = resource.Id
	rs.Resource = &resource
	rs.IsPublished = isPublished
	rs.LatestResourceChange = &latestResourceChange
	rs.EventMetadata = &eventMetadata

	return event.EventUnmarshaler{
		Version:     rs.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceStateSnapshotTaken{}),
		AggregateId: rs.Id,
		GroupId:     rs.Resource.DeviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceStateSnapshotTaken); ok {
				*x = *rs
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceUpdatePending(deviceId, resourceId string, content pb.Content, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
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
	return event.EventUnmarshaler{
		Version:     rc.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceUpdatePending{}),
		AggregateId: rc.Id,
		GroupId:     deviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdatePending); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceUpdated(deviceId, resourceId string, status pb.Status, content pb.Content, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
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
	return event.EventUnmarshaler{
		Version:     rc.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceUpdated{}),
		AggregateId: rc.Id,
		GroupId:     deviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdated); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceChangedEvent(id, deviceID string, content pb.Content, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
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
	return event.EventUnmarshaler{
		Version:     ru.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceChanged{}),
		AggregateId: ru.Id,
		GroupId:     deviceID,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceChanged); ok {
				*x = ru
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceRetrievePending(deviceId, resourceId string, resourceInterface string, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
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
	return event.EventUnmarshaler{
		Version:     rc.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceRetrievePending{}),
		AggregateId: rc.Id,
		GroupId:     deviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrievePending); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}

func MakeResourceRetrieved(deviceId, resourceId string, status pb.Status, content pb.Content, eventMetadata pb.EventMetadata) event.EventUnmarshaler {
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
	return event.EventUnmarshaler{
		Version:     rc.EventMetadata.Version,
		EventType:   httpUtils.ProtobufContentType(&pb.ResourceRetrieved{}),
		AggregateId: rc.Id,
		GroupId:     deviceId,
		Unmarshal: func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrieved); ok {
				*x = rc
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	}
}
