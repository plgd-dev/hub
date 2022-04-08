package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/strings"
)

const eventTypeResourceStateSnapshotTaken = "resourcestatesnapshottaken"

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
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceStateSnapshotTaken) CopyData(event *ResourceStateSnapshotTaken) {
	e.ResourceId = event.GetResourceId()
	e.LatestResourceChange = event.GetLatestResourceChange()
	e.ResourceCreatePendings = event.GetResourceCreatePendings()
	e.ResourceRetrievePendings = event.GetResourceRetrievePendings()
	e.ResourceUpdatePendings = event.GetResourceUpdatePendings()
	e.ResourceDeletePendings = event.GetResourceDeletePendings()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}

func (e *ResourceStateSnapshotTaken) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetLatestResourceChange() != nil &&
		e.GetResourceCreatePendings() != nil &&
		e.GetResourceRetrievePendings() != nil &&
		e.GetResourceUpdatePendings() != nil &&
		e.GetResourceDeletePendings() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

type resourceValidUntilValidator interface {
	ValidUntilTime() time.Time
	IsExpired(now time.Time) bool
	GetEventMetadata() *EventMetadata
	GetAuditContext() *commands.AuditContext
	GetResourceId() *commands.ResourceId
}

func (e *ResourceStateSnapshotTaken) processValidUntil(v resourceValidUntilValidator, now time.Time) (bool, error) {
	if v.IsExpired(now) {
		// for events from eventstore we just store metada from command.
		e.ResourceId = v.GetResourceId()
		e.EventMetadata = v.GetEventMetadata()
		e.AuditContext = v.GetAuditContext()
		return false, nil
	}
	return true, nil
}

func (e *ResourceStateSnapshotTaken) checkForDuplicitCorrelationID(correlationID string, now time.Time) error {
	for _, event := range e.GetResourceCreatePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == correlationID {
			return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource create pendings", correlationID)
		}
	}
	for _, event := range e.GetResourceUpdatePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == correlationID {
			return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource update pendings", correlationID)
		}
	}
	for _, event := range e.GetResourceRetrievePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == correlationID {
			return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource retrieve pendings", correlationID)
		}
	}
	for _, event := range e.GetResourceDeletePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == correlationID {
			return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource delete pendings", correlationID)
		}
	}
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceCreatePending(ctx context.Context, createPending *ResourceCreatePending) error {
	now := time.Now()
	ok, err := e.processValidUntil(createPending, now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	err = e.checkForDuplicitCorrelationID(createPending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = createPending.GetResourceId()
	e.EventMetadata = createPending.GetEventMetadata()
	e.ResourceCreatePendings = append(e.ResourceCreatePendings, createPending)
	e.AuditContext = createPending.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceUpdatePending(ctx context.Context, updatePending *ResourceUpdatePending) error {
	now := time.Now()
	ok, err := e.processValidUntil(updatePending, now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	err = e.checkForDuplicitCorrelationID(updatePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = updatePending.GetResourceId()
	e.EventMetadata = updatePending.GetEventMetadata()
	e.ResourceUpdatePendings = append(e.ResourceUpdatePendings, updatePending)
	e.AuditContext = updatePending.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceRetrievePending(ctx context.Context, retrievePending *ResourceRetrievePending) error {
	now := time.Now()
	ok, err := e.processValidUntil(retrievePending, now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	err = e.checkForDuplicitCorrelationID(retrievePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = retrievePending.GetResourceId()
	e.EventMetadata = retrievePending.GetEventMetadata()
	e.ResourceRetrievePendings = append(e.ResourceRetrievePendings, retrievePending)
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeletePending(ctx context.Context, deletePending *ResourceDeletePending) error {
	now := time.Now()
	ok, err := e.processValidUntil(deletePending, now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	err = e.checkForDuplicitCorrelationID(deletePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
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
	if e.GetLatestResourceChange().GetEventMetadata().GetConnectionId() != eventMetadata.GetConnectionId() {
		return true
	}
	if e.GetLatestResourceChange().GetEventMetadata().GetSequence() < eventMetadata.GetSequence() {
		return true
	}
	return false
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceChanged(ctx context.Context, changed *ResourceChanged) (bool, error) {
	if e.GetLatestResourceChange() == nil ||
		(e.ValidateSequence(changed.GetEventMetadata()) && !e.GetLatestResourceChange().Equal(changed)) {
		e.ResourceId = changed.GetResourceId()
		e.EventMetadata = changed.GetEventMetadata()
		e.LatestResourceChange = changed
		return true, nil
	}
	return false, nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceDeleted(ctx context.Context, deleted *ResourceDeleted) error {
	if deleted.GetStatus() == commands.Status_OK || deleted.GetStatus() == commands.Status_ACCEPTED {
		e.LatestResourceChange = nil
		e.ResourceCreatePendings = nil
		e.ResourceRetrievePendings = nil
		e.ResourceDeletePendings = nil
		e.ResourceUpdatePendings = nil
	} else {
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
		e.ResourceDeletePendings = append(e.ResourceDeletePendings[:index], e.ResourceDeletePendings[index+1:]...)
	}
	e.ResourceId = deleted.GetResourceId()
	e.EventMetadata = deleted.GetEventMetadata()
	e.AuditContext = deleted.GetAuditContext()
	return nil
}

func (e *ResourceStateSnapshotTaken) HandleEventResourceStateSnapshotTaken(ctx context.Context, snapshot *ResourceStateSnapshotTaken) {
	e.CopyData(snapshot)
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
			e.HandleEventResourceStateSnapshotTaken(ctx, &s)
		case (&ResourceUpdatePending{}).EventType():
			var s ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceUpdatePending(ctx, &s)
		case (&ResourceUpdated{}).EventType():
			var s ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceUpdated(ctx, &s)
		case (&ResourceCreatePending{}).EventType():
			var s ResourceCreatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceCreatePending(ctx, &s)
		case (&ResourceCreated{}).EventType():
			var s ResourceCreated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceCreated(ctx, &s)
		case (&ResourceChanged{}).EventType():
			var s ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_, _ = e.HandleEventResourceChanged(ctx, &s)
		case (&ResourceDeleted{}).EventType():
			var s ResourceDeleted
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceDeleted(ctx, &s)
		case (&ResourceDeletePending{}).EventType():
			var s ResourceDeletePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceDeletePending(ctx, &s)
		case (&ResourceRetrieved{}).EventType():
			var s ResourceRetrieved
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceRetrieved(ctx, &s)
		case (&ResourceRetrievePending{}).EventType():
			var s ResourceRetrievePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleEventResourceRetrievePending(ctx, &s)
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

func (e *ResourceStateSnapshotTaken) ConfirmResourceUpdateCommand(ctx context.Context, userID string, req *commands.ConfirmResourceUpdateRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.CommandMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
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
}

func (e *ResourceStateSnapshotTaken) ConfirmResourceRetrieveCommand(ctx context.Context, userID string, req *commands.ConfirmResourceRetrieveRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
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
}

func (e *ResourceStateSnapshotTaken) ConfirmResourceDeleteCommand(ctx context.Context, userID string, req *commands.ConfirmResourceDeleteRequest, newVersion uint64) ([]eventstore.Event, error) {
	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
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
}

func (e *ResourceStateSnapshotTaken) ConfirmResourceCreateCommand(ctx context.Context, userID string, req *commands.ConfirmResourceCreateRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
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

func (e *ResourceStateSnapshotTaken) CancelResourceCreateCommand(ctx context.Context, events []eventstore.Event, userID string, correlationIdFilter strings.Set, resourceID *commands.ResourceId, commandMetadata *commands.CommandMetadata, newVersion uint64) []eventstore.Event {
	for _, event := range e.GetResourceCreatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.ConfirmResourceCreateCommand(ctx, userID, &commands.ConfirmResourceCreateRequest{
			ResourceId:      resourceID,
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: commandMetadata,
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTaken) CancelResourceUpdateCommand(ctx context.Context, events []eventstore.Event, userID string, correlationIdFilter strings.Set, resourceID *commands.ResourceId, commandMetadata *commands.CommandMetadata, newVersion uint64) []eventstore.Event {
	for _, event := range e.GetResourceUpdatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.ConfirmResourceUpdateCommand(ctx, userID, &commands.ConfirmResourceUpdateRequest{
			ResourceId:      resourceID,
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: commandMetadata,
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTaken) CancelResourceRetrieveCommand(ctx context.Context, events []eventstore.Event, userID string, correlationIdFilter strings.Set, resourceID *commands.ResourceId, commandMetadata *commands.CommandMetadata, newVersion uint64) []eventstore.Event {
	for _, event := range e.GetResourceRetrievePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.ConfirmResourceRetrieveCommand(ctx, userID, &commands.ConfirmResourceRetrieveRequest{
			ResourceId:      resourceID,
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: commandMetadata,
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTaken) CancelResourceDeleteCommand(ctx context.Context, events []eventstore.Event, userID string, correlationIdFilter strings.Set, resourceID *commands.ResourceId, commandMetadata *commands.CommandMetadata, status commands.Status, content *commands.Content, newVersion uint64) []eventstore.Event {
	for _, event := range e.GetResourceDeletePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.ConfirmResourceDeleteCommand(ctx, userID, &commands.ConfirmResourceDeleteRequest{
			ResourceId:      resourceID,
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          status,
			CommandMetadata: commandMetadata,
			Content:         content,
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTaken) CancelPendingCommands(ctx context.Context, userID string, req *commands.CancelPendingCommandsRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	events := make([]eventstore.Event, 0, 4)
	events = e.CancelResourceCreateCommand(ctx, events, userID, correlationIdFilter, req.GetResourceId(), req.GetCommandMetadata(), newVersion)
	events = e.CancelResourceUpdateCommand(ctx, events, userID, correlationIdFilter, req.GetResourceId(), req.GetCommandMetadata(), newVersion)
	events = e.CancelResourceRetrieveCommand(ctx, events, userID, correlationIdFilter, req.GetResourceId(), req.GetCommandMetadata(), newVersion)
	events = e.CancelResourceDeleteCommand(ctx, events, userID, correlationIdFilter, req.GetResourceId(), req.GetCommandMetadata(), commands.Status_CANCELED, nil, newVersion)
	if len(events) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot find commands with correlationID(%v)", req.GetCorrelationIdFilter())
	}
	return events, nil
}

func (e *ResourceStateSnapshotTaken) UnpublishResource(ctx context.Context, userID string, resourceID *commands.ResourceId, commandMetadata *commands.CommandMetadata, deletedContent *commands.Content, newVersion uint64) ([]eventstore.Event, error) {
	if commandMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	correlationIdFilter := strings.MakeSet()
	events := make([]eventstore.Event, 0, 4)
	events = e.CancelResourceCreateCommand(ctx, events, userID, correlationIdFilter, resourceID, commandMetadata, newVersion)
	events = e.CancelResourceUpdateCommand(ctx, events, userID, correlationIdFilter, resourceID, commandMetadata, newVersion)
	events = e.CancelResourceRetrieveCommand(ctx, events, userID, correlationIdFilter, resourceID, commandMetadata, newVersion)
	events = e.CancelResourceDeleteCommand(ctx, events, userID, correlationIdFilter, resourceID, commandMetadata, commands.Status_OK, deletedContent, newVersion)
	snapshot, _ := e.TakeSnapshot(newVersion + uint64(len(events)))
	events = append(events, snapshot)

	return events, nil
}

func (e *ResourceStateSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return nil, err
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
		ac := commands.NewAuditContext(userID, "")

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
		ac := commands.NewAuditContext(userID, req.GetCorrelationId())
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
			ValidUntil:        timeToLive2ValidUntil(req.GetTimeToLive()),
		}

		if err = e.HandleEventResourceUpdatePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceUpdateRequest:
		return e.ConfirmResourceUpdateCommand(ctx, userID, req, newVersion)
	case *commands.RetrieveResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(userID, req.GetCorrelationId())

		rc := ResourceRetrievePending{
			ResourceId:        req.GetResourceId(),
			ResourceInterface: req.GetResourceInterface(),
			AuditContext:      ac,
			EventMetadata:     em,
			ValidUntil:        timeToLive2ValidUntil(req.GetTimeToLive()),
		}

		if err := e.HandleEventResourceRetrievePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceRetrieveRequest:
		return e.ConfirmResourceRetrieveCommand(ctx, userID, req, newVersion)
	case *commands.DeleteResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(userID, req.GetCorrelationId())

		rc := ResourceDeletePending{
			ResourceId:    req.GetResourceId(),
			AuditContext:  ac,
			EventMetadata: em,
			ValidUntil:    timeToLive2ValidUntil(req.GetTimeToLive()),
		}

		if err := e.HandleEventResourceDeletePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceDeleteRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}
		if req.GetStatus() == commands.Status_OK || req.GetStatus() == commands.Status_ACCEPTED {
			// deleted can comes first than unpublish so we call same handler
			return e.UnpublishResource(ctx, userID, req.GetResourceId(), req.GetCommandMetadata(), req.GetContent(), newVersion)
		}
		return e.ConfirmResourceDeleteCommand(ctx, userID, req, newVersion)
	case *commands.CreateResourceRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(userID, req.GetCorrelationId())
		content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
		if err != nil {
			return nil, err
		}
		rc := ResourceCreatePending{
			ResourceId:    req.GetResourceId(),
			Content:       content,
			AuditContext:  ac,
			EventMetadata: em,
			ValidUntil:    timeToLive2ValidUntil(req.GetTimeToLive()),
		}

		if err := e.HandleEventResourceCreatePending(ctx, &rc); err != nil {
			return nil, err
		}
		return []eventstore.Event{&rc}, nil
	case *commands.ConfirmResourceCreateRequest:
		return e.ConfirmResourceCreateCommand(ctx, userID, req, newVersion)
	case *commands.CancelPendingCommandsRequest:
		return e.CancelPendingCommands(ctx, userID, req, newVersion)
	case *commands.UnpublishResourceLinksRequest:
		return e.UnpublishResource(ctx, userID, e.GetResourceId(), req.GetCommandMetadata(), nil, newVersion)
	}

	return nil, fmt.Errorf("unknown command(%T)", cmd)
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
