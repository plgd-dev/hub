package events

import (
	"context"
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/net/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"
)

const eventTypeResourceStateSnapshotTaken = "ocf.cloud.resourceaggregate.events.resourcestatesnapshottaken"

const errInvalidDeviceID = "invalid device id"
const errInvalidVersion = "invalid version for events"
const errInvalidCommandMetadata = "invalid command metadata"

func (e *ResourceStateSnapshotTaken) AggregateId() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceStateSnapshotTaken) GroupId() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceStateSnapshotTaken) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceStateSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceStateSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceStateSnapshotTaken) EventType() string {
	return eventTypeResourceStateSnapshotTaken
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceUpdatePending(ctx context.Context, contentUpdatePending *ResourceUpdatePending) error {
	e.PendingRequestsCount++
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceRetrievePending(ctx context.Context, contentRetrievePending *ResourceRetrievePending) error {
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeletePending(ctx context.Context, req *ResourceDeletePending) error {
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceUpdated(ctx context.Context, contentUpdateProcessed *ResourceUpdated) error {
	e.PendingRequestsCount--
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceRetrieved(ctx context.Context, contentUpdateProcessed *ResourceRetrieved) error {
	return nil
}

func (e *ResourceStateSnapshotTaken) ValidateSequence(eventMetadata *EventMetadata) bool {
	if e.GetLatestResourceChange() == nil {
		return true
	}
	if e.GetLatestResourceChange().GetEventMetadata().GetConnectionId() != eventMetadata.GetConnectionId() {
		return true
	}
	if e.GetLatestResourceChange().GetEventMetadata().GetSequence() < eventMetadata.GetSequence() {
		return true
	}
	return false
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceChanged(ctx context.Context, contentChanged *ResourceChanged) (bool, error) {
	if e.ValidateSequence(contentChanged.GetEventMetadata()) {
		e.LatestResourceChange = contentChanged
		return true, nil
	}
	return false, nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeleted(ctx context.Context, deleted *ResourceDeleted) error {
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceStateSnapshotTaken(ctx context.Context, s *ResourceStateSnapshotTaken) error {
	if s.GetPendingRequestsCount() != 0 {
		return status.Errorf(codes.FailedPrecondition, "invalid pending requests")
	}
	e.ResourceId = s.GetResourceId()
	e.LatestResourceChange = s.GetLatestResourceChange()
	e.EventMetadata = s.GetEventMetadata()

	return nil
}

func (e *ResourceStateSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		switch eu.EventType() {
		case (&ResourceStateSnapshotTaken{}).EventType():
			var s ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceStateSnapshotTaken(ctx, &s); err != nil {
				return err
			}
		case (&ResourceUpdatePending{}).EventType():
			var s ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceUpdatePending(ctx, &s); err != nil {
				return err
			}
		case (&ResourceUpdated{}).EventType():
			var s ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceUpdated(ctx, &s); err != nil {
				return err
			}
		case (&ResourceChanged{}).EventType():
			var s ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if _, err := e.HandleEventResourceChanged(ctx, &s); err != nil {
				return err
			}
		case (&ResourceDeleted{}).EventType():
			var s ResourceDeleted
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceDeleted(ctx, &s); err != nil {
				return err
			}
		case (&ResourceRetrieved{}).EventType():
		case (&ResourceRetrievePending{}).EventType():
		}
	}
	return iter.Err()
}

func convertContent(content *commands.Content, supportedContentType string) (newContent *commands.Content, err error) {
	contentType := content.GetContentType()
	coapContentFormat := int32(-1)
	if content.GetCoapContentFormat() >= 0 && contentType == "" {
		contentType = message.MediaType(content.GetCoapContentFormat()).String()
	}
	if len(supportedContentType) == 0 {
		supportedContentType = contentType
	}
	var encode func(v interface{}) ([]byte, error)
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
	err = decode(content.GetData(), &m)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot decode content data from %v: %v", contentType, err)
	}

	data, err := encode(m)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot encode content data to %v: %v", message.MediaType(coapContentFormat).String(), err)
	}
	return &commands.Content{
		CoapContentFormat: coapContentFormat,
		ContentType:       message.MediaType(coapContentFormat).String(),
		Data:              data,
	}, nil
}

func (e *ResourceStateSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid userID: %v", err)
	}
	switch req := cmd.(type) {
	case *commands.NotifyResourceChangedRequest:
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")

		rc := ResourceChanged{
			ResourceId:    req.GetResourceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		var ok bool
		var err error
		if ok, err = e.HandleEventResourceChanged(ctx, &rc); err != nil {
			return nil, err
		}
		if ok {
			return []eventstore.Event{&rc}, nil
		}
		return nil, nil
	case *commands.UpdateResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())
		content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
		if err != nil {
			return nil, err
		}

		rc := ResourceUpdatePending{
			ResourceId:        req.GetResourceId(),
			ResourceInterface: req.GetResourceInterface(),
			AuditContext:      &ac,
			EventMetadata:     &em,
			Content:           content,
		}

		if err = e.HandleEventResourceUpdatePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceUpdateRequest:
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())
		rc := ResourceUpdated{
			ResourceId:    req.GetResourceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		if err := e.HandleEventResourceUpdated(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.RetrieveResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())

		rc := ResourceRetrievePending{
			ResourceId:        req.GetResourceId(),
			ResourceInterface: req.GetResourceInterface(),
			AuditContext:      &ac,
			EventMetadata:     &em,
		}

		if err := e.HandleEventResourceRetrievePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceRetrieveRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())
		rc := ResourceRetrieved{
			ResourceId:    req.GetResourceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		if err := e.HandleEventResourceRetrieved(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.DeleteResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())

		rc := ResourceDeletePending{
			ResourceId:    req.GetResourceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
		}

		if err := e.HandleEventResourceDeletePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceDeleteRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, req.GetCorrelationId())
		rc := ResourceDeleted{
			ResourceId:    req.GetResourceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		if err := e.HandleEventResourceDeleted(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (e *ResourceStateSnapshotTaken) SnapshotEventType() string { return e.EventType() }

func (e *ResourceStateSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	if e.PendingRequestsCount > 0 {
		return nil, false
	}
	e.EventMetadata.Version = version
	return e, true
}

func NewResourceStateSnapshotTaken() *ResourceStateSnapshotTaken {

	return &ResourceStateSnapshotTaken{
		EventMetadata: &EventMetadata{},
		//mapForCalculatePendingRequestsCount: make(map[string]bool),
	}
}
