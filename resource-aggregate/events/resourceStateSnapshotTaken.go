package events

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/go-coap/v2/message"

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

const errInvalidVersion = "invalid version for events"
const errInvalidCommandMetadata = "invalid command metadata"

func (e *ResourceStateSnapshotTaken) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceStateSnapshotTaken) GroupID() string {
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

func (e *ResourceStateSnapshotTaken) IsSnapshot() bool {
	return true
}

func (e *ResourceStateSnapshotTaken) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceCreatePending(ctx context.Context, createPending *ResourceCreatePending) error {
	for _, event := range e.GetResourceCreatePendings() {
		if event.GetAuditContext().GetCorrelationId() == createPending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "resource create pending with correlationId('%v') already exist", createPending.GetAuditContext().GetCorrelationId())
		}
	}
	e.ResourceId = createPending.GetResourceId()
	e.EventMetadata = createPending.GetEventMetadata()
	e.ResourceCreatePendings = append(e.ResourceCreatePendings, createPending)
	e.AuditContext = createPending.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceUpdatePending(ctx context.Context, updatePending *ResourceUpdatePending) error {
	for _, event := range e.GetResourceUpdatePendings() {
		if event.GetAuditContext().GetCorrelationId() == updatePending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "resource update pending with correlationId('%v') already exist", updatePending.GetAuditContext().GetCorrelationId())
		}
	}
	e.ResourceId = updatePending.GetResourceId()
	e.EventMetadata = updatePending.GetEventMetadata()
	e.ResourceUpdatePendings = append(e.ResourceUpdatePendings, updatePending)
	e.AuditContext = updatePending.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceRetrievePending(ctx context.Context, retrievePending *ResourceRetrievePending) error {
	for _, event := range e.GetResourceRetrievePendings() {
		if event.GetAuditContext().GetCorrelationId() == retrievePending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "resource retrieve pending with correlationId('%v') already exist", retrievePending.GetAuditContext().GetCorrelationId())
		}
	}
	e.ResourceId = retrievePending.GetResourceId()
	e.EventMetadata = retrievePending.GetEventMetadata()
	e.ResourceRetrievePendings = append(e.ResourceRetrievePendings, retrievePending)
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeletePending(ctx context.Context, deletePending *ResourceDeletePending) error {
	for _, event := range e.GetResourceDeletePendings() {
		if event.GetAuditContext().GetCorrelationId() == deletePending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "resource delete pending with correlationId('%v') already exist", deletePending.GetAuditContext().GetCorrelationId())
		}
	}
	e.ResourceId = deletePending.GetResourceId()
	e.EventMetadata = deletePending.GetEventMetadata()
	e.ResourceDeletePendings = append(e.ResourceDeletePendings, deletePending)
	e.AuditContext = deletePending.GetAuditContext()
	return nil
}
func RemoveIndex(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceCreated(ctx context.Context, created *ResourceCreated) error {
	index := -1
	for i, event := range e.GetResourceCreatePendings() {
		if event.GetAuditContext().GetCorrelationId() == created.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource create pending event with correlationId('%v')", created.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = created.GetResourceId()
	e.EventMetadata = created.GetEventMetadata()
	e.ResourceCreatePendings = append(e.ResourceCreatePendings[:index], e.ResourceCreatePendings[index+1:]...)
	e.AuditContext = created.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceUpdated(ctx context.Context, updated *ResourceUpdated) error {
	index := -1
	for i, event := range e.GetResourceUpdatePendings() {
		if event.GetAuditContext().GetCorrelationId() == updated.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource update pending event with correlationId('%v')", updated.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = updated.GetResourceId()
	e.EventMetadata = updated.GetEventMetadata()
	e.ResourceUpdatePendings = append(e.ResourceUpdatePendings[:index], e.ResourceUpdatePendings[index+1:]...)
	e.AuditContext = updated.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceRetrieved(ctx context.Context, retrieved *ResourceRetrieved) error {
	index := -1
	for i, event := range e.GetResourceRetrievePendings() {
		if event.GetAuditContext().GetCorrelationId() == retrieved.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource retrieve pending event with correlationId('%v')", retrieved.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = retrieved.GetResourceId()
	e.EventMetadata = retrieved.GetEventMetadata()
	e.ResourceRetrievePendings = append(e.ResourceRetrievePendings[:index], e.ResourceRetrievePendings[index+1:]...)
	e.AuditContext = retrieved.GetAuditContext()
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

func Equal(current, changed *ResourceChanged) bool {
	if current.GetStatus() != changed.GetStatus() {
		return false
	}

	if current.GetContent().GetCoapContentFormat() != changed.GetContent().GetCoapContentFormat() ||
		current.GetContent().GetContentType() != changed.GetContent().GetContentType() ||
		!bytes.Equal(current.GetContent().GetData(), changed.GetContent().GetData()) {
		return false
	}

	if current.GetAuditContext().GetUserId() != changed.GetAuditContext().GetUserId() {
		return false
	}

	return true
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceChanged(ctx context.Context, changed *ResourceChanged) (bool, error) {
	if e.ValidateSequence(changed.GetEventMetadata()) &&
		(e.LatestResourceChange == nil || !Equal(e.LatestResourceChange, changed)) {
		e.ResourceId = changed.GetResourceId()
		e.EventMetadata = changed.GetEventMetadata()
		e.LatestResourceChange = changed
		return true, nil
	}
	return false, nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeleted(ctx context.Context, deleted *ResourceDeleted) error {
	index := -1
	for i, event := range e.GetResourceDeletePendings() {
		if event.GetAuditContext().GetCorrelationId() == deleted.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource delete pending event with correlationId('%v')", deleted.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = deleted.GetResourceId()
	e.EventMetadata = deleted.GetEventMetadata()
	e.ResourceDeletePendings = append(e.ResourceDeletePendings[:index], e.ResourceDeletePendings[index+1:]...)
	e.AuditContext = deleted.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceStateSnapshotTaken(ctx context.Context, snapshot *ResourceStateSnapshotTaken) error {
	e.ResourceId = snapshot.GetResourceId()
	e.LatestResourceChange = snapshot.GetLatestResourceChange()
	e.EventMetadata = snapshot.GetEventMetadata()

	e.ResourceCreatePendings = snapshot.GetResourceCreatePendings()
	e.ResourceRetrievePendings = snapshot.GetResourceRetrievePendings()
	e.ResourceUpdatePendings = snapshot.GetResourceUpdatePendings()
	e.ResourceDeletePendings = snapshot.GetResourceDeletePendings()

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
		case (&ResourceCreatePending{}).EventType():
			var s ResourceCreatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceCreatePending(ctx, &s); err != nil {
				return err
			}
		case (&ResourceCreated{}).EventType():
			var s ResourceCreated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceCreated(ctx, &s); err != nil {
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
		case (&ResourceDeletePending{}).EventType():
			var s ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceDeletePending(ctx, &s); err != nil {
				return err
			}
		case (&ResourceRetrieved{}).EventType():
			var s ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceRetrieved(ctx, &s); err != nil {
				return err
			}
		case (&ResourceRetrievePending{}).EventType():
			var s ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceRetrievePending(ctx, &s); err != nil {
				return err
			}
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
	owner, err := grpc.OwnerFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner: %v", err)
	}

	// only NotifyResourceChangedRequest can have version 0
	if _, ok := cmd.(*commands.NotifyResourceChangedRequest); !ok && newVersion == 0 {
		return nil, status.Errorf(codes.NotFound, errInvalidVersion)
	}

	switch req := cmd.(type) {
	case *commands.NotifyResourceChangedRequest:
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(owner, "")

		rc := ResourceChanged{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
		if err != nil {
			return nil, err
		}

		rc := ResourceUpdatePending{
			ResourceId:        req.GetResourceId(),
			ResourceInterface: req.GetResourceInterface(),
			AuditContext:      ac,
			EventMetadata:     em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		rc := ResourceUpdated{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())

		rc := ResourceRetrievePending{
			ResourceId:        req.GetResourceId(),
			ResourceInterface: req.GetResourceInterface(),
			AuditContext:      ac,
			EventMetadata:     em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		rc := ResourceRetrieved{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())

		rc := ResourceDeletePending{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
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
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		rc := ResourceDeleted{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		if err := e.HandleEventResourceDeleted(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.CreateResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
		if err != nil {
			return nil, err
		}
		rc := ResourceCreatePending{
			ResourceId:    req.GetResourceId(),
			Content:       content,
			AuditContext:  ac,
			EventMetadata: em,
		}

		if err := e.HandleEventResourceCreatePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceCreateRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		rc := ResourceCreated{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
			Content:       req.GetContent(),
			Status:        req.GetStatus(),
		}
		if err := e.HandleEventResourceCreated(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (e *ResourceStateSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &ResourceStateSnapshotTaken{
		ResourceId:               e.GetResourceId(),
		LatestResourceChange:     e.GetLatestResourceChange(),
		EventMetadata:            MakeEventMeta(e.GetEventMetadata().GetConnectionId(), e.GetEventMetadata().GetSequence(), version),
		ResourceCreatePendings:   e.GetResourceCreatePendings(),
		ResourceUpdatePendings:   e.GetResourceUpdatePendings(),
		ResourceRetrievePendings: e.GetResourceRetrievePendings(),
		ResourceDeletePendings:   e.GetResourceDeletePendings(),
		AuditContext:             e.GetAuditContext(),
	}, true
}

func NewResourceStateSnapshotTaken() *ResourceStateSnapshotTaken {
	return &ResourceStateSnapshotTaken{
		EventMetadata: &EventMetadata{},
	}
}
