package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataSnapshotTaken = "devicemetadatasnapshottaken"

func (e *DeviceMetadataSnapshotTaken) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *DeviceMetadataSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *DeviceMetadataSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *DeviceMetadataSnapshotTaken) EventType() string {
	return eventTypeDeviceMetadataSnapshotTaken
}

func (e *DeviceMetadataSnapshotTaken) AggregateID() string {
	return commands.MakeStatusResourceUUID(e.GetDeviceId())
}

func (e *DeviceMetadataSnapshotTaken) GroupID() string {
	return e.GetDeviceId()
}

func (e *DeviceMetadataSnapshotTaken) IsSnapshot() bool {
	return true
}

func (e *DeviceMetadataSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *DeviceMetadataSnapshotTaken) CopyData(event *DeviceMetadataSnapshotTaken) {
	e.DeviceId = event.GetDeviceId()
	e.DeviceMetadataUpdated = event.GetDeviceMetadataUpdated()
	e.UpdatePendings = event.GetUpdatePendings()
	e.EventMetadata = event.GetEventMetadata()
}

func (e *DeviceMetadataSnapshotTaken) CheckInitialized() bool {
	return e.GetDeviceId() != "" &&
		e.GetDeviceMetadataUpdated() != nil &&
		e.GetUpdatePendings() != nil &&
		e.GetEventMetadata() != nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdated(ctx context.Context, upd *DeviceMetadataUpdated, confirm bool) (bool, error) {
	index := -1
	for i, event := range e.GetUpdatePendings() {
		if event.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if confirm && index < 0 {
		return false, status.Errorf(codes.InvalidArgument, "cannot find shadow synchronization status update pending event with correlationId('%v')", upd.GetAuditContext().GetCorrelationId())
	}
	if e.DeviceMetadataUpdated.Equal(upd) {
		return false, nil
	}
	e.DeviceId = upd.GetDeviceId()
	if index >= 0 {
		e.UpdatePendings = append(e.UpdatePendings[:index], e.UpdatePendings[index+1:]...)
	}
	e.DeviceMetadataUpdated = upd
	e.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(ctx context.Context, s *DeviceMetadataSnapshotTaken) {
	e.CopyData(s)
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdatePending(ctx context.Context, updatePending *DeviceMetadataUpdatePending) error {
	now := time.Now()
	if updatePending.IsExpired(now) {
		e.DeviceId = updatePending.GetDeviceId()
		e.EventMetadata = updatePending.GetEventMetadata()
		// for events from eventstore we do nothing
		return nil
	}
	for _, event := range e.GetUpdatePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == updatePending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "device metadata update pending with correlationId('%v') already exist", updatePending.GetAuditContext().GetCorrelationId())
		}
	}
	e.DeviceId = updatePending.GetDeviceId()
	e.EventMetadata = updatePending.GetEventMetadata()
	e.UpdatePendings = append(e.UpdatePendings, updatePending)
	return nil
}

func (e *DeviceMetadataSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		switch eu.EventType() {
		case (&DeviceMetadataSnapshotTaken{}).EventType():
			var s DeviceMetadataSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			e.HandleDeviceMetadataSnapshotTaken(ctx, &s)
		case (&DeviceMetadataUpdated{}).EventType():
			var s DeviceMetadataUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_, _ = e.HandleDeviceMetadataUpdated(ctx, &s, false)
		case (&DeviceMetadataUpdatePending{}).EventType():
			var s DeviceMetadataUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = e.HandleDeviceMetadataUpdatePending(ctx, &s)
		}
	}
	return iter.Err()
}

func timeToLive2ValidUntil(timeToLive int64) int64 {
	if timeToLive == 0 {
		return 0
	}
	return pkgTime.UnixNano(time.Now().Add(time.Duration(timeToLive)))
}

func (e *DeviceMetadataSnapshotTaken) ConfirmDeviceMetadataUpdate(ctx context.Context, userID string, req *commands.ConfirmDeviceMetadataUpdateRequest, newVersion uint64, cancel bool) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
	switch {
	case cancel:
		ev := DeviceMetadataUpdated{
			DeviceId:              req.GetDeviceId(),
			Status:                e.GetDeviceMetadataUpdated().GetStatus(),
			ShadowSynchronization: e.GetDeviceMetadataUpdated().GetShadowSynchronization(),
			Canceled:              true,
			AuditContext:          ac,
			EventMetadata:         em,
		}
		ok, err := e.HandleDeviceMetadataUpdated(ctx, &ev, true)
		if !ok {
			return nil, err
		}
		return []eventstore.Event{&ev}, nil
	case req.GetShadowSynchronization() != commands.ShadowSynchronization_UNSET:
		ev := DeviceMetadataUpdated{
			DeviceId:              req.GetDeviceId(),
			Status:                e.GetDeviceMetadataUpdated().GetStatus(),
			ShadowSynchronization: req.GetShadowSynchronization(),
			AuditContext:          ac,
			EventMetadata:         em,
		}
		ok, err := e.HandleDeviceMetadataUpdated(ctx, &ev, true)
		if !ok {
			return nil, err
		}
		return []eventstore.Event{&ev}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown confirm type(%T)", req.GetConfirm())
	}
}

func (e *DeviceMetadataSnapshotTaken) CancelPendingMetadataUpdates(ctx context.Context, userID string, req *commands.CancelPendingMetadataUpdatesRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	events := make([]eventstore.Event, 0, 4)
	for _, event := range e.GetUpdatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := e.ConfirmDeviceMetadataUpdate(ctx, userID, &commands.ConfirmDeviceMetadataUpdateRequest{
			DeviceId:        req.GetDeviceId(),
			CorrelationId:   event.GetAuditContext().GetCorrelationId(),
			Status:          commands.Status_CANCELED,
			CommandMetadata: req.GetCommandMetadata(),
		}, newVersion+uint64(len(events)), true)
		if err == nil {
			// errors appears only when command with correlationID doesn't exist
			events = append(events, ev...)
		}
	}
	if len(events) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot find commands with correlationID(%v)", req.GetCorrelationIdFilter())
	}
	return events, nil
}

func (e *DeviceMetadataSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return nil, err
	}

	switch req := cmd.(type) {
	case *commands.UpdateDeviceMetadataRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(userID, req.GetCorrelationId())

		switch {
		case req.GetStatus() != nil:
			// it is expected that the device updates the status on its own. no confirmation needed.
			ev := DeviceMetadataUpdated{
				DeviceId:              req.GetDeviceId(),
				Status:                req.GetStatus(),
				ShadowSynchronization: e.GetDeviceMetadataUpdated().GetShadowSynchronization(),
				AuditContext:          ac,
				EventMetadata:         em,
			}
			ok, err := e.HandleDeviceMetadataUpdated(ctx, &ev, false)
			if !ok {
				return nil, err
			}
			return []eventstore.Event{&ev}, nil
		case req.GetShadowSynchronization() != commands.ShadowSynchronization_UNSET:
			ev := DeviceMetadataUpdatePending{
				DeviceId:   req.GetDeviceId(),
				ValidUntil: timeToLive2ValidUntil(req.GetTimeToLive()),
				UpdatePending: &DeviceMetadataUpdatePending_ShadowSynchronization{
					ShadowSynchronization: req.GetShadowSynchronization(),
				},
				AuditContext:  ac,
				EventMetadata: em,
			}
			err := e.HandleDeviceMetadataUpdatePending(ctx, &ev)
			if err != nil {
				return nil, err
			}
			return []eventstore.Event{&ev}, nil
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown update type(%T)", req.GetUpdate())
		}
	case *commands.ConfirmDeviceMetadataUpdateRequest:
		return e.ConfirmDeviceMetadataUpdate(ctx, userID, req, newVersion, false)
	case *commands.CancelPendingMetadataUpdatesRequest:
		return e.CancelPendingMetadataUpdates(ctx, userID, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (e *DeviceMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &DeviceMetadataSnapshotTaken{
		DeviceId:              e.GetDeviceId(),
		EventMetadata:         MakeEventMeta(e.GetEventMetadata().GetConnectionId(), e.GetEventMetadata().GetSequence(), version),
		DeviceMetadataUpdated: e.GetDeviceMetadataUpdated(),
	}, true
}

func NewDeviceMetadataSnapshotTaken() *DeviceMetadataSnapshotTaken {
	return &DeviceMetadataSnapshotTaken{
		EventMetadata: &EventMetadata{},
	}
}
