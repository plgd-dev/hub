package events

import (
	"context"
	"fmt"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/net/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cqrsUtils "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	"github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/cqrs"
	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/net/http"
)

type VerifyAccessFunc func(deviceId, resourceId string) error

type ResourceStateSnapshotTaken struct {
	pb.ResourceStateSnapshotTaken
	mapForCalculatePendingRequestsCount map[string]bool
	verifyAccess                        VerifyAccessFunc
}

func (rs *ResourceStateSnapshotTaken) AggregateId() string {
	return rs.Id
}

func (rs *ResourceStateSnapshotTaken) GroupId() string {
	return rs.Resource.DeviceId
}

func (rs *ResourceStateSnapshotTaken) Version() uint64 {
	return rs.ResourceStateSnapshotTaken.EventMetadata.Version
}

func (rs *ResourceStateSnapshotTaken) Marshal() ([]byte, error) {
	return rs.ResourceStateSnapshotTaken.Marshal()
}

func (rs *ResourceStateSnapshotTaken) Unmarshal(b []byte) error {
	return rs.ResourceStateSnapshotTaken.Unmarshal(b)
}

func (rs *ResourceStateSnapshotTaken) EventType() string {
	return http.ProtobufContentType(&pb.ResourceStateSnapshotTaken{})
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourcePublished(ctx context.Context, pub ResourcePublished) error {
	rs.Id = pub.Resource.Id
	if rs.IsPublished {
		pub.Resource.InstanceId = rs.Resource.InstanceId
	}
	rs.Resource = pub.Resource
	rs.TimeToLive = pub.TimeToLive
	rs.IsPublished = true
	return nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceUnpublished(ctx context.Context, pub ResourceUnpublished) error {
	if !rs.IsPublished {
		return status.Errorf(codes.FailedPrecondition, "resource is already unpublished")
	}
	rs.IsPublished = false
	return nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceUpdatePending(ctx context.Context, contentUpdatePending ResourceUpdatePending) error {
	rs.mapForCalculatePendingRequestsCount[contentUpdatePending.AuditContext.CorrelationId] = true
	return nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceRetrievePending(ctx context.Context, contentRetrievePending ResourceRetrievePending) error {
	if !rs.IsPublished {
		return status.Errorf(codes.FailedPrecondition, "resource is unpublished")
	}
	return nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceUpdated(ctx context.Context, contentUpdateProcessed ResourceUpdated) error {
	delete(rs.mapForCalculatePendingRequestsCount, contentUpdateProcessed.AuditContext.CorrelationId)
	rs.PendingRequestsCount = uint32(len(rs.mapForCalculatePendingRequestsCount))
	return nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceRetrieved(ctx context.Context, contentUpdateProcessed ResourceRetrieved) error {
	return nil
}

func (rs *ResourceStateSnapshotTaken) ValidateSequence(eventMetadata *pb.EventMetadata) bool {
	if rs.LatestResourceChange == nil {
		return true
	}
	if rs.GetLatestResourceChange().GetEventMetadata().GetConnectionId() != eventMetadata.GetConnectionId() {
		return true
	}
	if rs.GetLatestResourceChange().GetEventMetadata().GetSequence() < eventMetadata.GetSequence() {
		return true
	}
	return false
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceChanged(ctx context.Context, contentChanged ResourceChanged) (bool, error) {
	if rs.ValidateSequence(contentChanged.EventMetadata) {
		rs.LatestResourceChange = &contentChanged.ResourceChanged
		return true, nil
	}
	return false, nil
}

func (rs *ResourceStateSnapshotTaken) HandleEventResourceStateSnapshotTaken(ctx context.Context, s ResourceStateSnapshotTaken) error {
	if s.PendingRequestsCount != 0 {
		return status.Errorf(codes.FailedPrecondition, "invalid pending requests")
	}
	rs.Id = s.Resource.Id
	rs.Resource = s.Resource
	rs.LatestResourceChange = s.LatestResourceChange
	rs.TimeToLive = s.TimeToLive
	rs.IsPublished = s.IsPublished
	rs.EventMetadata = s.EventMetadata

	return nil
}

func (rs *ResourceStateSnapshotTaken) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler
	for iter.Next(ctx, &eu) {
		if eu.EventType == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		err := rs.verifyAccess(eu.GroupId, eu.AggregateId)
		if err != nil {
			return grpc.ForwardErrorf(codes.Unauthenticated, "unauthorized access to resource: %v", err)
		}
		switch eu.EventType {
		case http.ProtobufContentType(&pb.ResourceStateSnapshotTaken{}):
			var s ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rs.HandleEventResourceStateSnapshotTaken(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourcePublished{}):
			var s ResourcePublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rs.HandleEventResourcePublished(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceUnpublished{}):
			var s ResourceUnpublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rs.HandleEventResourceUnpublished(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceUpdatePending{}):
			var s ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rs.HandleEventResourceUpdatePending(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceUpdated{}):
			var s ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rs.HandleEventResourceUpdated(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceChanged{}):
			var s ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if _, err := rs.HandleEventResourceChanged(ctx, s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceRetrieved{}):
		case http.ProtobufContentType(&pb.ResourceRetrievePending{}):
		}
	}
	return iter.Err()
}

func convertContent(content *pb.Content, supportedContentTypes []string) (newContent *pb.Content, err error) {
	contentType := content.ContentType
	coapContentFormat := int32(-1)
	if len(supportedContentTypes) == 0 {
		supportedContentTypes = []string{message.AppOcfCbor.String()}
	}
	if content.CoapContentFormat >= 0 && contentType == "" {
		contentType = message.MediaType(content.CoapContentFormat).String()
	}
	var encode func(v interface{}) ([]byte, error)
	for _, supportedContentType := range supportedContentTypes {
		switch supportedContentType {
		case contentType:
			return content, nil
		case message.AppCBOR.String():
			encode = cbor.Encode
			coapContentFormat = int32(message.AppCBOR)
		case message.AppOcfCbor.String():
			if encode == nil {
				encode = cbor.Encode
				coapContentFormat = int32(message.AppOcfCbor)
			}
		case message.AppJSON.String():
			if encode == nil {
				encode = json.Encode
				coapContentFormat = int32(message.AppJSON)
			}
		}

	}

	if encode == nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot convert content-type from %v: unknown target", contentType)
	}

	var decode func(in []byte, v interface{}) error
	switch contentType {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decode = cbor.Decode
	case message.AppJSON.String():
		decode = json.Decode
	default:
		return nil, status.Errorf(codes.InvalidArgument, "cannot convert content-type from %v: unsupported source", contentType)
	}

	var m interface{}
	err = decode(content.Data, &m)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot decode content data from %v: %v", contentType, err)
	}

	data, err := encode(m)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot encode content data to %v: %v", message.MediaType(coapContentFormat).String(), err)
	}
	return &pb.Content{
		CoapContentFormat: coapContentFormat,
		ContentType:       message.MediaType(coapContentFormat).String(),
		Data:              data,
	}, nil
}

func (rs *ResourceStateSnapshotTaken) HandleCommand(ctx context.Context, cmd cqrs.Command, newVersion uint64) ([]event.Event, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot handle command: invalid userID: %w", err)
	}
	switch req := cmd.(type) {
	case *pb.PublishResourceRequest:
		if rs.Id != req.ResourceId && rs.Id != "" {
			return nil, status.Errorf(codes.Internal, "cannot handle resource publish: invalid resource id")
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle resource publish: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")

		rp := ResourcePublished{pb.ResourcePublished{
			Id:            req.ResourceId,
			Resource:      req.Resource,
			TimeToLive:    req.TimeToLive,
			AuditContext:  &ac,
			EventMetadata: &em,
		},
		}
		err := rs.HandleEventResourcePublished(ctx, rp)
		if err != nil {
			return nil, fmt.Errorf("cannot handle resource publish: %w", err)
		}
		return []event.Event{rp}, nil
	case *pb.UnpublishResourceRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle resource unpublish: invalid version for events")
		}
		if rs.Id != req.ResourceId {
			return nil, status.Errorf(codes.Internal, "cannot handle resource unpublish: invalid resource id")
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle resource unpublish: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")
		ru := ResourceUnpublished{pb.ResourceUnpublished{
			Id:            req.ResourceId,
			AuditContext:  &ac,
			EventMetadata: &em,
		}}
		err := rs.HandleEventResourceUnpublished(ctx, ru)
		if err != nil {
			return nil, fmt.Errorf("cannot handle resource unpublish: %w", err)
		}
		return []event.Event{ru}, nil
	case *pb.NotifyResourceChangedRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle notify resource changed: invalid version for events")
		}
		if rs.Id != req.ResourceId {
			return nil, status.Errorf(codes.Internal, "cannot handle notify resource changed: invalid resource id")
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle notify resource changed: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")

		rc := ResourceChanged{
			pb.ResourceChanged{
				Id:            req.ResourceId,
				AuditContext:  &ac,
				EventMetadata: &em,
				Content:       req.Content,
				Status:        req.Status,
			},
		}
		var ok bool
		var err error
		if ok, err = rs.HandleEventResourceChanged(ctx, rc); err != nil {
			return nil, fmt.Errorf("cannot handle notify resource changed: %w", err)
		}
		if ok {
			return []event.Event{rc}, nil
		}
		return nil, nil
	case *pb.UpdateResourceRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle update resource content invalid version for events")
		}
		if rs.Id != req.GetResourceId() {
			return nil, status.Errorf(codes.Internal, "cannot handle update resource content: invalid resource id")
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle update resource content: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.CorrelationId)
		content, err := convertContent(req.Content, rs.Resource.SupportedContentTypes)
		if err != nil {
			return nil, fmt.Errorf("cannot handle notify resource changed: %w", err)
		}

		rc := ResourceUpdatePending{
			pb.ResourceUpdatePending{
				Id:                req.GetResourceId(),
				ResourceInterface: req.GetResourceInterface(),
				AuditContext:      &ac,
				EventMetadata:     &em,
				Content:           content,
			},
		}

		if err = rs.HandleEventResourceUpdatePending(ctx, rc); err != nil {
			return nil, fmt.Errorf("cannot handle update resource content: %w", err)
		}
		return []event.Event{rc}, nil
	case *pb.ConfirmResourceUpdateRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle notify resource content update processed: invalid version for events")
		}
		if rs.Id != req.ResourceId {
			return nil, status.Errorf(codes.Internal, "cannot handle notify resource content update processed: invalid resource id")
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle notify resource content update processed: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.CorrelationId)
		rc := ResourceUpdated{
			pb.ResourceUpdated{
				Id:            req.ResourceId,
				AuditContext:  &ac,
				EventMetadata: &em,
				Content:       req.Content,
				Status:        req.Status,
			},
		}
		if err := rs.HandleEventResourceUpdated(ctx, rc); err != nil {
			return nil, fmt.Errorf("cannot handle notify resource content update processed: %w", err)
		}
		return []event.Event{rc}, nil
	case *pb.RetrieveResourceRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle retrieve resource content invalid version for events")
		}
		if rs.Id != req.GetResourceId() {
			return nil, status.Errorf(codes.Internal, "cannot handle retrieve resource content: invalid resource id")
		}
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle retrieve resource content: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.CorrelationId)

		rc := ResourceRetrievePending{
			pb.ResourceRetrievePending{
				Id:                req.GetResourceId(),
				ResourceInterface: req.GetResourceInterface(),
				AuditContext:      &ac,
				EventMetadata:     &em,
			},
		}

		if err := rs.HandleEventResourceRetrievePending(ctx, rc); err != nil {
			return nil, fmt.Errorf("cannot handle retrieve resource content: %w", err)
		}
		return []event.Event{rc}, nil
	case *pb.ConfirmResourceRetrieveRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, "cannot handle notify resource content retrieve processed: invalid version for events")
		}
		if rs.Id != req.GetResourceId() {
			return nil, status.Errorf(codes.Internal, "cannot handle notify resource content retrieve processed: invalid resource id")
		}
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot handle notify resource content retrieve processed: invalid command metadata")
		}

		em := cqrsUtils.MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := cqrsUtils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())
		rc := ResourceRetrieved{
			pb.ResourceRetrieved{
				Id:            req.GetResourceId(),
				AuditContext:  &ac,
				EventMetadata: &em,
				Content:       req.GetContent(),
				Status:        req.GetStatus(),
			},
		}
		if err := rs.HandleEventResourceRetrieved(ctx, rc); err != nil {
			return nil, fmt.Errorf("cannot handle notify resource content retrieve processed: %w", err)
		}
		return []event.Event{rc}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (rs *ResourceStateSnapshotTaken) SnapshotEventType() string { return rs.EventType() }

func (rs *ResourceStateSnapshotTaken) TakeSnapshot(version uint64) (event.Event, bool) {
	if rs.PendingRequestsCount > 0 {
		return nil, false
	}
	rs.EventMetadata.Version = version
	return rs, true
}

func NewResourceStateSnapshotTaken(verifyAccess VerifyAccessFunc) *ResourceStateSnapshotTaken {

	return &ResourceStateSnapshotTaken{
		ResourceStateSnapshotTaken: pb.ResourceStateSnapshotTaken{
			Resource:      &pb.Resource{},
			EventMetadata: &pb.EventMetadata{},
		},
		mapForCalculatePendingRequestsCount: make(map[string]bool),
		verifyAccess:                        verifyAccess,
	}
}
