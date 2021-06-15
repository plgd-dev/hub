package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataSnapshotTaken = "ocf.cloud.resourceaggregate.events.devicemetadatasnapshottaken"

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
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdated(ctx context.Context, upd *DeviceMetadataUpdated, confirm bool) (bool, error) {
	if e.DeviceMetadataUpdated.Equal(upd) {
		return false, nil
	}
	e.DeviceId = upd.GetDeviceId()
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
	if index >= 0 {
		e.UpdatePendings = append(e.UpdatePendings[:index], e.UpdatePendings[index+1:]...)
	}
	e.DeviceMetadataUpdated = upd
	e.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(ctx context.Context, s *DeviceMetadataSnapshotTaken) error {
	e.DeviceId = s.GetDeviceId()
	e.DeviceMetadataUpdated = s.GetDeviceMetadataUpdated()
	e.EventMetadata = s.GetEventMetadata()
	return nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdatePending(ctx context.Context, updatePending *DeviceMetadataUpdatePending) error {
	for _, event := range e.GetUpdatePendings() {
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
			if err := e.HandleDeviceMetadataSnapshotTaken(ctx, &s); err != nil {
				return err
			}
		case (&DeviceMetadataUpdated{}).EventType():
			var s DeviceMetadataUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if _, err := e.HandleDeviceMetadataUpdated(ctx, &s, false); err != nil {
				return err
			}
		case (&DeviceMetadataUpdatePending{}).EventType():
			var s DeviceMetadataUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleDeviceMetadataUpdatePending(ctx, &s); err != nil {
				return err
			}
		}
	}
	return iter.Err()
}

func (e *DeviceMetadataSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	owner, err := grpc.OwnerFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner: %v", err)
	}
	switch req := cmd.(type) {
	case *commands.UpdateDeviceMetadataRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())

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
				DeviceId: req.GetDeviceId(),
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
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.NewAuditContext(owner, req.GetCorrelationId())
		switch {
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

	return nil, fmt.Errorf("unknown command")
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
