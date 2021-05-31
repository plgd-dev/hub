package events

import (
	"context"
	"fmt"

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

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdated(ctx context.Context, upd *DeviceMetadataUpdated) error {
	e.DeviceId = upd.GetDeviceId()
	if upd.GetStatus() != nil {
		e.Status = upd.GetStatus()
	}
	if upd.GetShadowSynchronization() != nil {
		index := -1
		for i, event := range e.GetUpdatePendings() {
			if event.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId() {
				index = i
				break
			}
		}
		if index < 0 {
			return status.Errorf(codes.InvalidArgument, "cannot find shadow synchronization status update pending event with correlationId('%v')", upd.GetAuditContext().GetCorrelationId())
		}
		e.UpdatePendings = append(e.UpdatePendings[:index], e.UpdatePendings[index+1:]...)
		e.ShadowSynchronization = upd.GetShadowSynchronization()
	}
	e.EventMetadata = upd.GetEventMetadata()
	return nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(ctx context.Context, s *DeviceMetadataSnapshotTaken) error {
	e.DeviceId = s.GetDeviceId()
	e.ShadowSynchronization = s.GetShadowSynchronization()
	e.Status = s.GetStatus()
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
			if err := e.HandleDeviceMetadataUpdated(ctx, &s); err != nil {
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
				DeviceId: req.GetDeviceId(),
				Updated: &DeviceMetadataUpdated_Status{
					Status: req.GetStatus(),
				},
				AuditContext:  ac,
				EventMetadata: em,
			}
			err := e.HandleDeviceMetadataUpdated(ctx, &ev)
			if err != nil {
				return nil, err
			}
			return []eventstore.Event{&ev}, nil
		case req.GetShadowSynchronization() != nil:
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
		case req.GetShadowSynchronization() != nil:
			ev := DeviceMetadataUpdated{
				DeviceId: req.GetDeviceId(),
				Updated: &DeviceMetadataUpdated_ShadowSynchronization{
					ShadowSynchronization: req.GetShadowSynchronization(),
				},
				AuditContext:  ac,
				EventMetadata: em,
			}
			err := e.HandleDeviceMetadataUpdated(ctx, &ev)
			if err != nil {
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
	e.EventMetadata.Version = version
	return &DeviceMetadataSnapshotTaken{
		DeviceId:              e.GetDeviceId(),
		EventMetadata:         e.GetEventMetadata(),
		Status:                e.GetStatus(),
		ShadowSynchronization: e.GetShadowSynchronization(),
	}, true
}

func NewDeviceMetadataSnapshotTaken() *DeviceMetadataSnapshotTaken {
	return &DeviceMetadataSnapshotTaken{
		EventMetadata: &EventMetadata{},
	}
}
