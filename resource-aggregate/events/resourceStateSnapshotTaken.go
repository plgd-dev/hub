package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceStateSnapshotTaken = "resourcestatesnapshottaken"

const (
	errInvalidVersion           = "invalid version for events"
	errResourceChangedNotExists = "resource changed not exists"
	errInvalidCommandMetadata   = "invalid command metadata"
)

func (e *ResourceStateSnapshotTaken) AggregateID() string {
	return e.GetResourceId().ToUUID().String()
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

func (e *ResourceStateSnapshotTaken) ETag() *eventstore.ETagData {
	return e.GetLatestResourceChange().ETag()
}

func (e *ResourceStateSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceStateSnapshotTaken) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceStateSnapshotTaken) Types() []string {
	return e.GetResourceTypes()
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
	e.ResourceTypes = event.GetResourceTypes()
}

func (e *ResourceStateSnapshotTaken) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil && (e.GetLatestResourceChange() != nil ||
		e.GetResourceCreatePendings() != nil ||
		e.GetResourceRetrievePendings() != nil ||
		e.GetResourceUpdatePendings() != nil ||
		e.GetResourceDeletePendings() != nil)
}

type resourceValidUntilValidator interface {
	ValidUntilTime() time.Time
	IsExpired(now time.Time) bool
	GetEventMetadata() *EventMetadata
	GetAuditContext() *commands.AuditContext
	GetResourceId() *commands.ResourceId
}

func (e *ResourceStateSnapshotTaken) processValidUntil(v resourceValidUntilValidator, now time.Time) bool {
	if v.IsExpired(now) {
		// for events from eventstore we just store metada from command.
		e.ResourceId = v.GetResourceId()
		e.EventMetadata = v.GetEventMetadata()
		e.AuditContext = v.GetAuditContext()
		return false
	}
	return true
}

type pendingEvent interface {
	GetAuditContext() *commands.AuditContext
	IsExpired(now time.Time) bool
}

func checkForDuplicityCorrelationID[T pendingEvent](pes []T, correlationID string, now time.Time) bool {
	for _, pe := range pes {
		if pe.IsExpired(now) {
			continue
		}
		if pe.GetAuditContext().GetCorrelationId() == correlationID {
			return false
		}
	}
	return true
}

func (e *ResourceStateSnapshotTaken) checkForDuplicityCorrelationID(correlationID string, now time.Time) error {
	if ok := checkForDuplicityCorrelationID(e.GetResourceCreatePendings(), correlationID, now); !ok {
		return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource create pendings", correlationID)
	}
	if ok := checkForDuplicityCorrelationID(e.GetResourceUpdatePendings(), correlationID, now); !ok {
		return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource update pendings", correlationID)
	}
	if ok := checkForDuplicityCorrelationID(e.GetResourceRetrievePendings(), correlationID, now); !ok {
		return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource retrieve pendings", correlationID)
	}
	if ok := checkForDuplicityCorrelationID(e.GetResourceDeletePendings(), correlationID, now); !ok {
		return status.Errorf(codes.InvalidArgument, "duplicit correlationId('%v') at resource delete pendings", correlationID)
	}
	return nil
}

func (e *ResourceStateSnapshotTaken) handleEventResourceCreatePending(createPending *ResourceCreatePending) error {
	now := time.Now()
	if ok := e.processValidUntil(createPending, now); !ok {
		return nil
	}
	err := e.checkForDuplicityCorrelationID(createPending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = createPending.GetResourceId()
	e.EventMetadata = createPending.GetEventMetadata()
	e.ResourceCreatePendings = append(e.ResourceCreatePendings, createPending)
	e.AuditContext = createPending.GetAuditContext()
	e.setResourceTypes(createPending.GetResourceTypes())
	return nil
}

func (e *ResourceStateSnapshotTaken) setResourceTypes(resourceTypes []string) {
	if len(resourceTypes) == 0 {
		return
	}
	e.ResourceTypes = resourceTypes
}

func (e *ResourceStateSnapshotTaken) handleEventResourceUpdatePending(updatePending *ResourceUpdatePending) error {
	now := time.Now()
	if ok := e.processValidUntil(updatePending, now); !ok {
		return nil
	}
	err := e.checkForDuplicityCorrelationID(updatePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = updatePending.GetResourceId()
	e.EventMetadata = updatePending.GetEventMetadata()
	e.ResourceUpdatePendings = append(e.ResourceUpdatePendings, updatePending)
	e.AuditContext = updatePending.GetAuditContext()
	e.setResourceTypes(updatePending.GetResourceTypes())
	return nil
}

func (e *ResourceStateSnapshotTaken) handleEventResourceRetrievePending(retrievePending *ResourceRetrievePending) error {
	now := time.Now()
	if ok := e.processValidUntil(retrievePending, now); !ok {
		return nil
	}
	err := e.checkForDuplicityCorrelationID(retrievePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = retrievePending.GetResourceId()
	e.EventMetadata = retrievePending.GetEventMetadata()
	e.ResourceRetrievePendings = append(e.ResourceRetrievePendings, retrievePending)
	e.AuditContext = retrievePending.GetAuditContext()
	e.setResourceTypes(retrievePending.GetResourceTypes())
	return nil
}

func (e *ResourceStateSnapshotTaken) handleEventResourceDeletePending(deletePending *ResourceDeletePending) error {
	now := time.Now()
	if ok := e.processValidUntil(deletePending, now); !ok {
		return nil
	}
	err := e.checkForDuplicityCorrelationID(deletePending.GetAuditContext().GetCorrelationId(), now)
	if err != nil {
		return err
	}
	e.ResourceId = deletePending.GetResourceId()
	e.EventMetadata = deletePending.GetEventMetadata()
	e.ResourceDeletePendings = append(e.ResourceDeletePendings, deletePending)
	e.AuditContext = deletePending.GetAuditContext()
	e.setResourceTypes(deletePending.GetResourceTypes())
	return nil
}

func RemoveIndex(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}

func (e *ResourceStateSnapshotTaken) handleEventResourceCreated(created *ResourceCreated) error {
	resourceCreatePendings := e.GetResourceCreatePendings()
	index := findResourceOperationPendingIndex(created.GetAuditContext().GetCorrelationId(), resourceCreatePendings)
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource create pending event with correlationId('%v')", created.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = created.GetResourceId()
	e.EventMetadata = created.GetEventMetadata()
	resourceCreatePendings = append(resourceCreatePendings[:index], resourceCreatePendings[index+1:]...)
	e.ResourceCreatePendings = resourceCreatePendings
	e.AuditContext = created.GetAuditContext()
	e.setResourceTypes(created.GetResourceTypes())
	return nil
}

func findResourceOperationPendingIndex[Op interface{ GetAuditContext() *commands.AuditContext }](correlationID string, ops []Op) int {
	for i, event := range ops {
		if event.GetAuditContext().GetCorrelationId() == correlationID {
			return i
		}
	}
	return -1
}

func (e *ResourceStateSnapshotTaken) handleEventResourceUpdated(updated *ResourceUpdated) error {
	resourceUpdatePendings := e.GetResourceUpdatePendings()
	index := findResourceOperationPendingIndex(updated.GetAuditContext().GetCorrelationId(), resourceUpdatePendings)
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource update pending event with correlationId('%v')", updated.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = updated.GetResourceId()
	e.EventMetadata = updated.GetEventMetadata()
	resourceUpdatePendings = append(resourceUpdatePendings[:index], resourceUpdatePendings[index+1:]...)
	e.ResourceUpdatePendings = resourceUpdatePendings
	e.AuditContext = updated.GetAuditContext()
	e.setResourceTypes(updated.GetResourceTypes())
	return nil
}

func (e *ResourceStateSnapshotTaken) handleEventResourceRetrieved(retrieved *ResourceRetrieved) error {
	resourceRetrievePendings := e.GetResourceRetrievePendings()
	index := findResourceOperationPendingIndex(retrieved.GetAuditContext().GetCorrelationId(), resourceRetrievePendings)
	if index < 0 {
		return status.Errorf(codes.InvalidArgument, "cannot find resource retrieve pending event with correlationId('%v')", retrieved.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = retrieved.GetResourceId()
	e.EventMetadata = retrieved.GetEventMetadata()
	resourceRetrievePendings = append(resourceRetrievePendings[:index], resourceRetrievePendings[index+1:]...)
	e.ResourceRetrievePendings = resourceRetrievePendings
	e.AuditContext = retrieved.GetAuditContext()
	e.setResourceTypes(retrieved.GetResourceTypes())
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

func (e *ResourceStateSnapshotTaken) handleEventResourceChanged(changed *ResourceChanged) bool {
	if e.GetLatestResourceChange() == nil ||
		(e.ValidateSequence(changed.GetEventMetadata()) && !e.GetLatestResourceChange().Equal(changed)) {
		e.ResourceId = changed.GetResourceId()
		e.EventMetadata = changed.GetEventMetadata()
		e.LatestResourceChange = changed
		e.AuditContext = changed.GetAuditContext()
		e.setResourceTypes(changed.GetResourceTypes())
		return true
	}
	return false
}

func (e *ResourceStateSnapshotTaken) findResourceDeletePendingIndex(status commands.Status, correlationID string) (bool, int) {
	if status == commands.Status_OK || status == commands.Status_ACCEPTED {
		return true, -1
	}
	return false, findResourceOperationPendingIndex(correlationID, e.GetResourceDeletePendings())
}

func (e *ResourceStateSnapshotTaken) handleEventResourceDeleted(deleted *ResourceDeleted) error {
	deleteResource, deletePendingIndex := e.findResourceDeletePendingIndex(deleted.GetStatus(), deleted.GetAuditContext().GetCorrelationId())
	switch {
	case deleteResource:
		e.ResourceCreatePendings = nil
		e.ResourceRetrievePendings = nil
		e.ResourceDeletePendings = nil
		e.ResourceUpdatePendings = nil
	case deletePendingIndex >= 0:
		resourceDeletePendings := e.GetResourceDeletePendings()
		resourceDeletePendings = append(resourceDeletePendings[:deletePendingIndex], resourceDeletePendings[deletePendingIndex+1:]...)
		e.ResourceDeletePendings = resourceDeletePendings
	default:
		return status.Errorf(codes.InvalidArgument, "cannot find resource delete pending event with correlationId('%v')", deleted.GetAuditContext().GetCorrelationId())
	}
	e.ResourceId = deleted.GetResourceId()
	e.EventMetadata = deleted.GetEventMetadata()
	e.AuditContext = deleted.GetAuditContext()
	e.setResourceTypes(deleted.GetResourceTypes())
	return nil
}

func (e *ResourceStateSnapshotTaken) handleEventResourceStateSnapshotTaken(snapshot *ResourceStateSnapshotTaken) {
	e.CopyData(snapshot)
}

//nolint:gocyclo
func (e *ResourceStateSnapshotTaken) handleByEvent(eu eventstore.EventUnmarshaler) error {
	switch eu.EventType() {
	case (&ResourceStateSnapshotTaken{}).EventType():
		var s ResourceStateSnapshotTaken
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		e.handleEventResourceStateSnapshotTaken(&s)
	case (&ResourceUpdatePending{}).EventType():
		var s ResourceUpdatePending
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceUpdatePending(&s)
	case (&ResourceUpdated{}).EventType():
		var s ResourceUpdated
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceUpdated(&s)
	case (&ResourceCreatePending{}).EventType():
		var s ResourceCreatePending
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceCreatePending(&s)
	case (&ResourceCreated{}).EventType():
		var s ResourceCreated
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceCreated(&s)
	case (&ResourceChanged{}).EventType():
		var s ResourceChanged
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceChanged(&s)
	case (&ResourceDeleted{}).EventType():
		var s ResourceDeleted
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceDeleted(&s)
	case (&ResourceDeletePending{}).EventType():
		var s ResourceDeletePending
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceDeletePending(&s)
	case (&ResourceRetrieved{}).EventType():
		var s ResourceRetrieved
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceRetrieved(&s)
	case (&ResourceRetrievePending{}).EventType():
		var s ResourceRetrievePending
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		_ = e.handleEventResourceRetrievePending(&s)
	}
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
		if err := e.handleByEvent(eu); err != nil {
			return err
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
		encode = cbor.Encode
		coapContentFormat = int32(message.AppOcfCbor)
	case message.AppJSON.String():
		encode = json.Encode
		coapContentFormat = int32(message.AppJSON)
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

func (e *ResourceStateSnapshotTakenForCommand) confirmResourceUpdateRequest(ctx context.Context, req *commands.ConfirmResourceUpdateRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	rc := ResourceUpdated{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              req.GetContent(),
		Status:               req.GetStatus(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}
	if err := e.handleEventResourceUpdated(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) confirmResourceRetrieveRequest(ctx context.Context, req *commands.ConfirmResourceRetrieveRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	rc := ResourceRetrieved{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              req.GetContent(),
		Status:               req.GetStatus(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		Etag:                 req.GetEtag(),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}
	if err := e.handleEventResourceRetrieved(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) confirmResourceDeleteRequest(ctx context.Context, req *commands.ConfirmResourceDeleteRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	rc := ResourceDeleted{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              req.GetContent(),
		Status:               req.GetStatus(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}
	if err := e.handleEventResourceDeleted(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) confirmResourceCreateRequest(ctx context.Context, req *commands.ConfirmResourceCreateRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	rc := ResourceCreated{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              req.GetContent(),
		Status:               req.GetStatus(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}

	if err := e.handleEventResourceCreated(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) cancelResourceCreatePendings(ctx context.Context, req *commands.CancelPendingCommandsRequest, newVersion uint64) []eventstore.Event {
	events := make([]eventstore.Event, 0, 4)
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	for _, event := range e.GetResourceCreatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.confirmResourceCreateRequest(ctx, &commands.ConfirmResourceCreateRequest{
			ResourceId:      req.GetResourceId(),
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: req.GetCommandMetadata(),
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTakenForCommand) cancelResourceUpdatePendings(ctx context.Context, req *commands.CancelPendingCommandsRequest, newVersion uint64) []eventstore.Event {
	events := make([]eventstore.Event, 0, 4)
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	for _, event := range e.GetResourceUpdatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.confirmResourceUpdateRequest(ctx, &commands.ConfirmResourceUpdateRequest{
			ResourceId:      req.GetResourceId(),
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: req.GetCommandMetadata(),
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTakenForCommand) cancelResourceRetrievePendings(ctx context.Context, req *commands.CancelPendingCommandsRequest, newVersion uint64) []eventstore.Event {
	events := make([]eventstore.Event, 0, 4)
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	for _, event := range e.GetResourceRetrievePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.confirmResourceRetrieveRequest(ctx, &commands.ConfirmResourceRetrieveRequest{
			ResourceId:      req.GetResourceId(),
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: req.GetCommandMetadata(),
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTakenForCommand) cancelResourceDeletePendings(ctx context.Context, req *commands.CancelPendingCommandsRequest, newVersion uint64) []eventstore.Event {
	events := make([]eventstore.Event, 0, 4)
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	for _, event := range e.GetResourceDeletePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.confirmResourceDeleteRequest(ctx, &commands.ConfirmResourceDeleteRequest{
			ResourceId:      req.GetResourceId(),
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: req.GetCommandMetadata(),
		}, newVersion+uint64(len(events)))
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	return events
}

func (e *ResourceStateSnapshotTakenForCommand) CancelPendingCommandsRequest(ctx context.Context, req *commands.CancelPendingCommandsRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	events := e.cancelResourceCreatePendings(ctx, req, newVersion)
	events = append(events, e.cancelResourceUpdatePendings(ctx, req, newVersion+uint64(len(events)))...)
	events = append(events, e.cancelResourceRetrievePendings(ctx, req, newVersion+uint64(len(events)))...)
	events = append(events, e.cancelResourceDeletePendings(ctx, req, newVersion+uint64(len(events)))...)
	if len(events) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot find commands with correlationID(%v)", req.GetCorrelationIdFilter())
	}
	return events, nil
}

func (e *ResourceStateSnapshotTakenForCommand) handleNotifyResourceChangedRequest(ctx context.Context, req *commands.NotifyResourceChangedRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, "", e.owner)

	rc := ResourceChanged{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              req.GetContent(),
		Status:               req.GetStatus(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		Etag:                 req.GetEtag(),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), req.GetResourceTypes()),
	}

	if e.handleEventResourceChanged(&rc) {
		return []eventstore.Event{&rc}, nil
	}
	return nil, nil
}

func (e *ResourceStateSnapshotTakenForCommand) handleUpdateResourceRequest(ctx context.Context, req *commands.UpdateResourceRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
	if err != nil {
		return nil, err
	}

	rc := ResourceUpdatePending{
		ResourceId:           req.GetResourceId(),
		ResourceInterface:    req.GetResourceInterface(),
		AuditContext:         ac,
		EventMetadata:        em,
		Content:              content,
		ValidUntil:           timeToLive2ValidUntil(req.GetTimeToLive()),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}

	if err = e.handleEventResourceUpdatePending(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) handleRetrieveResourceRequest(ctx context.Context, req *commands.RetrieveResourceRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)

	rc := ResourceRetrievePending{
		ResourceId:           req.GetResourceId(),
		ResourceInterface:    req.GetResourceInterface(),
		AuditContext:         ac,
		EventMetadata:        em,
		ValidUntil:           timeToLive2ValidUntil(req.GetTimeToLive()),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		Etag:                 req.GetEtag(),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}

	if err := e.handleEventResourceRetrievePending(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) handleDeleteResourceRequest(ctx context.Context, req *commands.DeleteResourceRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)

	rc := ResourceDeletePending{
		ResourceId:           req.GetResourceId(),
		AuditContext:         ac,
		EventMetadata:        em,
		ValidUntil:           timeToLive2ValidUntil(req.GetTimeToLive()),
		ResourceInterface:    req.GetResourceInterface(),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}

	if err := e.handleEventResourceDeletePending(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) resolveResourceTypes(href string, resourceTypes []string) []string {
	if len(resourceTypes) > 0 {
		// resourceTypes from command has higher priority
		return resourceTypes
	}
	if e.resourceLinks == nil {
		// if no resourceLinks, return resourceTypes from snapshot
		return e.GetResourceTypes()
	}
	resources := e.resourceLinks.GetResources()
	if len(resources) == 0 {
		// if no resourceLinks, return resourceTypes from snapshot
		return e.GetResourceTypes()
	}
	link, ok := e.resourceLinks.GetResources()[href]
	if !ok {
		// if resourceLinks doesn't contain resource, return resourceTypes from snapshot
		return e.GetResourceTypes()
	}
	// return resourceTypes from link
	return link.GetResourceTypes()
}

func (e *ResourceStateSnapshotTakenForCommand) handleCreateResourceRequest(ctx context.Context, req *commands.CreateResourceRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
	ac := commands.NewAuditContext(e.userID, req.GetCorrelationId(), e.owner)
	content, err := convertContent(req.GetContent(), e.GetLatestResourceChange().GetContent().GetContentType())
	if err != nil {
		return nil, err
	}
	rc := ResourceCreatePending{
		ResourceId:           req.GetResourceId(),
		Content:              content,
		AuditContext:         ac,
		EventMetadata:        em,
		ValidUntil:           timeToLive2ValidUntil(req.GetTimeToLive()),
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		ResourceTypes:        e.resolveResourceTypes(req.GetResourceId().GetHref(), nil),
	}

	if err := e.handleEventResourceCreatePending(&rc); err != nil {
		return nil, err
	}
	return []eventstore.Event{&rc}, nil
}

func (e *ResourceStateSnapshotTakenForCommand) validateCancelPendingCommandsForNotExistingResource(req *commands.CancelPendingCommandsRequest) bool {
	if len(e.GetResourceUpdatePendings()) == 0 && len(e.GetResourceCreatePendings()) == 0 && len(e.GetResourceDeletePendings()) == 0 {
		return false
	}
	if len(req.GetCorrelationIdFilter()) == 0 {
		return true
	}
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	for _, event := range e.GetResourceUpdatePendings() {
		if correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			return true
		}
	}
	for _, event := range e.GetResourceCreatePendings() {
		if correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			return true
		}
	}
	for _, event := range e.GetResourceDeletePendings() {
		if correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			return true
		}
	}
	return false
}

func (e *ResourceStateSnapshotTakenForCommand) validateCommandForNotExistingResource(cmd aggregate.Command) bool {
	if e.GetLatestResourceChange() != nil {
		// resource exists
		return true
	}
	switch req := cmd.(type) {
	case *commands.NotifyResourceChangedRequest:
		// NotifyResourceChangedRequest can have any version
		return true
	case *commands.UpdateResourceRequest:
		// UpdateResourceRequest can have version 0 when if not exists is set
		return req.GetForce()
	case *commands.ConfirmResourceUpdateRequest:
		return findResourceOperationPendingIndex(req.GetCorrelationId(), e.GetResourceUpdatePendings()) >= 0
	case *commands.CreateResourceRequest:
		// CreateResourceRequest can have version 0 when if not exists is set
		return req.GetForce()
	case *commands.ConfirmResourceCreateRequest:
		return findResourceOperationPendingIndex(req.GetCorrelationId(), e.GetResourceCreatePendings()) >= 0
	case *commands.DeleteResourceRequest:
		// DeleteResourceRequest can have version 0 when if not exists is set
		return req.GetForce()
	case *commands.ConfirmResourceDeleteRequest:
		deleteResource, deletePending := e.findResourceDeletePendingIndex(req.GetStatus(), req.GetCorrelationId())
		return deleteResource || deletePending >= 0
	case *commands.CancelPendingCommandsRequest:
		return e.validateCancelPendingCommandsForNotExistingResource(req)
	}
	return false
}

func (e *ResourceStateSnapshotTakenForCommand) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	if !e.validateCommandForNotExistingResource(cmd) {
		return nil, status.Errorf(codes.NotFound, errResourceChangedNotExists)
	}

	switch req := cmd.(type) {
	case *commands.NotifyResourceChangedRequest:
		return e.handleNotifyResourceChangedRequest(ctx, req, newVersion)
	case *commands.UpdateResourceRequest:
		return e.handleUpdateResourceRequest(ctx, req, newVersion)
	case *commands.ConfirmResourceUpdateRequest:
		return e.confirmResourceUpdateRequest(ctx, req, newVersion)
	case *commands.RetrieveResourceRequest:
		return e.handleRetrieveResourceRequest(ctx, req, newVersion)
	case *commands.ConfirmResourceRetrieveRequest:
		return e.confirmResourceRetrieveRequest(ctx, req, newVersion)
	case *commands.DeleteResourceRequest:
		return e.handleDeleteResourceRequest(ctx, req, newVersion)
	case *commands.ConfirmResourceDeleteRequest:
		return e.confirmResourceDeleteRequest(ctx, req, newVersion)
	case *commands.CreateResourceRequest:
		return e.handleCreateResourceRequest(ctx, req, newVersion)
	case *commands.ConfirmResourceCreateRequest:
		return e.confirmResourceCreateRequest(ctx, req, newVersion)
	case *commands.CancelPendingCommandsRequest:
		return e.CancelPendingCommandsRequest(ctx, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command(%T)", cmd)
}

func (e *ResourceStateSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &ResourceStateSnapshotTaken{
		ResourceId:               e.GetResourceId(),
		LatestResourceChange:     e.GetLatestResourceChange(),
		EventMetadata:            MakeEventMeta(e.GetEventMetadata().GetConnectionId(), e.GetEventMetadata().GetSequence(), version, e.GetEventMetadata().GetHubId()),
		ResourceCreatePendings:   e.GetResourceCreatePendings(),
		ResourceUpdatePendings:   e.GetResourceUpdatePendings(),
		ResourceRetrievePendings: e.GetResourceRetrievePendings(),
		ResourceDeletePendings:   e.GetResourceDeletePendings(),
		AuditContext:             e.GetAuditContext(),
		ResourceTypes:            e.GetResourceTypes(),
	}, true
}

type ResourceStateSnapshotTakenForCommand struct {
	owner         string
	hubID         string
	userID        string
	resourceLinks *ResourceLinksSnapshotTakenForCommand
	*ResourceStateSnapshotTaken
}

func NewResourceStateSnapshotTakenForCommand(userID string, owner string, hubID string, resourceLinks *ResourceLinksSnapshotTakenForCommand) *ResourceStateSnapshotTakenForCommand {
	return &ResourceStateSnapshotTakenForCommand{
		ResourceStateSnapshotTaken: NewResourceStateSnapshotTaken(),
		userID:                     userID,
		owner:                      owner,
		hubID:                      hubID,
		resourceLinks:              resourceLinks,
	}
}

func NewResourceStateSnapshotTaken() *ResourceStateSnapshotTaken {
	return &ResourceStateSnapshotTaken{
		EventMetadata: &EventMetadata{},
	}
}
