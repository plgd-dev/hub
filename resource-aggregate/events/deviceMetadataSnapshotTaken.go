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

const eventTypeDeviceMetadataSnapshotTaken = "ocf.cloud.resourceaggregate.events.devicecloudstatussnapshottaken"

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
	if upd.GetOnline() != nil {
		e.Online = upd.GetOnline()
	}
	if upd.GetShadowSynchronizationStatus() != nil {
		e.ShadowSynchronizationStatus = upd.GetShadowSynchronizationStatus()
	}
	e.EventMetadata = upd.GetEventMetadata()
	return nil
}

func (e *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(ctx context.Context, s *DeviceMetadataSnapshotTaken) error {
	e.DeviceId = s.GetDeviceId()
	e.ShadowSynchronizationStatus = s.GetShadowSynchronizationStatus()
	e.Online = s.GetOnline()
	e.EventMetadata = s.GetEventMetadata()
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
		ac := commands.NewAuditContext(owner, "")

		var v isDeviceMetadataUpdated_Changed
		switch {
		case req.GetOnline() != nil:
			v = &DeviceMetadataUpdated_Online{
				Online: req.GetOnline(),
			}
		case req.GetShadowSynchronizationStatus() != nil:
			v = &DeviceMetadataUpdated_ShadowSynchronizationStatus{
				ShadowSynchronizationStatus: req.GetShadowSynchronizationStatus(),
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown update type(%T)", req.GetUpdate())
		}
		rlp := DeviceMetadataUpdated{
			DeviceId:      req.GetDeviceId(),
			Changed:       v,
			AuditContext:  ac,
			EventMetadata: em,
		}
		err := e.HandleDeviceMetadataUpdated(ctx, &rlp)
		if err != nil {
			return nil, err
		}
		return []eventstore.Event{&rlp}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (e *DeviceMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	e.EventMetadata.Version = version
	return &DeviceMetadataSnapshotTaken{
		DeviceId:                    e.GetDeviceId(),
		EventMetadata:               e.GetEventMetadata(),
		Online:                      e.GetOnline(),
		ShadowSynchronizationStatus: e.GetShadowSynchronizationStatus(),
	}, true
}

func NewDeviceMetadataSnapshotTaken() *DeviceMetadataSnapshotTaken {
	return &DeviceMetadataSnapshotTaken{
		EventMetadata: &EventMetadata{},
	}
}
