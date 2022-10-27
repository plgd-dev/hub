package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataSnapshotTaken = "devicemetadatasnapshottaken"

func (d *DeviceMetadataSnapshotTaken) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *DeviceMetadataSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *DeviceMetadataSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *DeviceMetadataSnapshotTaken) EventType() string {
	return eventTypeDeviceMetadataSnapshotTaken
}

func (d *DeviceMetadataSnapshotTaken) AggregateID() string {
	return commands.MakeStatusResourceUUID(d.GetDeviceId())
}

func (d *DeviceMetadataSnapshotTaken) GroupID() string {
	return d.GetDeviceId()
}

func (d *DeviceMetadataSnapshotTaken) IsSnapshot() bool {
	return true
}

func (d *DeviceMetadataSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *DeviceMetadataSnapshotTaken) CopyData(event *DeviceMetadataSnapshotTaken) {
	d.DeviceId = event.GetDeviceId()
	d.DeviceMetadataUpdated = event.GetDeviceMetadataUpdated()
	d.UpdatePendings = event.GetUpdatePendings()
	d.EventMetadata = event.GetEventMetadata()
}

func (d *DeviceMetadataSnapshotTaken) CheckInitialized() bool {
	return d.GetDeviceId() != "" &&
		d.GetDeviceMetadataUpdated() != nil &&
		d.GetUpdatePendings() != nil &&
		d.GetEventMetadata() != nil
}

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdated(ctx context.Context, upd *DeviceMetadataUpdated, confirm bool) (bool, error) {
	index := -1
	for i, event := range d.GetUpdatePendings() {
		if event.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId() {
			index = i
			break
		}
	}
	if confirm && index < 0 {
		return false, status.Errorf(codes.InvalidArgument, "cannot find twin synchronization update pending event with correlationId('%v')", upd.GetAuditContext().GetCorrelationId())
	}
	if index >= 0 {
		d.UpdatePendings = append(d.UpdatePendings[:index], d.UpdatePendings[index+1:]...)
	}
	if d.DeviceMetadataUpdated.Equal(upd) {
		return false, nil
	}
	if d.DeviceMetadataUpdated.GetConnection().IsOnline() && upd.GetConnection() != nil && !upd.GetConnection().IsOnline() && d.DeviceMetadataUpdated.GetConnection().GetId() != upd.GetConnection().GetId() {
		// if previous status was online and new status is offline, the connectionId must be the same
		return false, nil
	}
	d.DeviceId = upd.GetDeviceId()
	if d.DeviceMetadataUpdated == nil {
		d.DeviceMetadataUpdated = upd
	} else {
		d.DeviceMetadataUpdated.CopyData(upd)
	}
	d.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(ctx context.Context, s *DeviceMetadataSnapshotTaken) {
	d.CopyData(s)
}

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdatePending(ctx context.Context, updatePending *DeviceMetadataUpdatePending) error {
	now := time.Now()
	if updatePending.IsExpired(now) {
		d.DeviceId = updatePending.GetDeviceId()
		d.EventMetadata = updatePending.GetEventMetadata()
		// for events from eventstore we do nothing
		return nil
	}
	for _, event := range d.GetUpdatePendings() {
		if event.IsExpired(now) {
			continue
		}
		if event.GetAuditContext().GetCorrelationId() == updatePending.GetAuditContext().GetCorrelationId() {
			return status.Errorf(codes.InvalidArgument, "device metadata update pending with correlationId('%v') already exist", updatePending.GetAuditContext().GetCorrelationId())
		}
	}
	d.DeviceId = updatePending.GetDeviceId()
	d.EventMetadata = updatePending.GetEventMetadata()
	d.UpdatePendings = append(d.UpdatePendings, updatePending)
	return nil
}

func (d *DeviceMetadataSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
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
			d.HandleDeviceMetadataSnapshotTaken(ctx, &s)
		case (&DeviceMetadataUpdated{}).EventType():
			var s DeviceMetadataUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_, _ = d.HandleDeviceMetadataUpdated(ctx, &s, false)
		case (&DeviceMetadataUpdatePending{}).EventType():
			var s DeviceMetadataUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_ = d.HandleDeviceMetadataUpdatePending(ctx, &s)
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

func (d *DeviceMetadataSnapshotTaken) ConfirmDeviceMetadataUpdate(ctx context.Context, userID string, req *commands.ConfirmDeviceMetadataUpdateRequest, newVersion uint64, cancel bool) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
	_, is_confirm_twin_enabled := req.GetConfirm().(*commands.ConfirmDeviceMetadataUpdateRequest_TwinEnabled)
	switch {
	case cancel:
		ev := DeviceMetadataUpdated{
			DeviceId:             req.GetDeviceId(),
			Connection:           d.GetDeviceMetadataUpdated().GetConnection(),
			TwinSynchronization:  d.GetDeviceMetadataUpdated().GetTwinSynchronization(),
			TwinEnabled:          d.GetDeviceMetadataUpdated().GetTwinEnabled(),
			Canceled:             true,
			AuditContext:         ac,
			EventMetadata:        em,
			OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		}
		ok, err := d.HandleDeviceMetadataUpdated(ctx, &ev, true)
		if !ok {
			return nil, err
		}
		return []eventstore.Event{&ev}, nil
	case is_confirm_twin_enabled:
		twinSynchronization := d.GetDeviceMetadataUpdated().GetTwinSynchronization()
		if !req.GetTwinEnabled() && req.GetCommandMetadata().GetConnectionId() == d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetConnectionId() {
			// reset twinSynchronization
			if twinSynchronization != nil {
				twinSynchronization.State = commands.TwinSynchronization_NONE
			}
		}
		ev := DeviceMetadataUpdated{
			DeviceId:             req.GetDeviceId(),
			Connection:           d.GetDeviceMetadataUpdated().GetConnection(),
			TwinEnabled:          req.GetTwinEnabled(),
			TwinSynchronization:  twinSynchronization,
			AuditContext:         ac,
			EventMetadata:        em,
			OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		}
		ok, err := d.HandleDeviceMetadataUpdated(ctx, &ev, true)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
		return []eventstore.Event{&ev}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown confirm type(%T)", req.GetConfirm())
	}
}

func (d *DeviceMetadataSnapshotTaken) CancelPendingMetadataUpdates(ctx context.Context, userID string, req *commands.CancelPendingMetadataUpdatesRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	events := make([]eventstore.Event, 0, 4)
	for _, event := range d.GetUpdatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := d.ConfirmDeviceMetadataUpdate(ctx, userID, &commands.ConfirmDeviceMetadataUpdateRequest{
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

func (d *DeviceMetadataSnapshotTaken) updateDeviceConnection(ctx context.Context, req *commands.UpdateDeviceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	req.GetConnection().Id = req.GetCommandMetadata().GetConnectionId()
	if req.GetConnection().GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update connection status for empty connectionId")
	}
	// it is expected that the device updates the status on its own. no confirmation needed.

	// keep last connected at from the previous event
	lastConnectedAt := d.GetDeviceMetadataUpdated().GetConnection().GetConnectedAt()
	if req.GetConnection().GetConnectedAt() < lastConnectedAt {
		req.GetConnection().ConnectedAt = lastConnectedAt
	}

	twinSynchronization := d.GetDeviceMetadataUpdated().GetTwinSynchronization()
	// check if it is new connection
	if req.GetConnection().GetStatus() == commands.Connection_ONLINE && req.GetConnection().GetId() != d.GetDeviceMetadataUpdated().GetConnection().GetId() {
		// reset twinSynchronization
		if twinSynchronization == nil {
			twinSynchronization = &commands.TwinSynchronization{}
		}
		twinSynchronization.State = commands.TwinSynchronization_NONE
		twinSynchronization.CommandMetadata = req.GetCommandMetadata()
	}
	ev := DeviceMetadataUpdated{
		DeviceId:             req.GetDeviceId(),
		Connection:           req.GetConnection(),
		TwinEnabled:          d.GetDeviceMetadataUpdated().GetTwinEnabled(),
		TwinSynchronization:  twinSynchronization,
		AuditContext:         ac,
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
	}
	ok, err := d.HandleDeviceMetadataUpdated(ctx, &ev, false)
	if !ok {
		return nil, err
	}
	return []eventstore.Event{&ev}, nil
}

func (d *DeviceMetadataSnapshotTaken) updateDeviceTwinSynchronization(ctx context.Context, req *commands.UpdateDeviceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	commandMetadata := req.GetCommandMetadata()
	// it is expected that the device updates the status on its own. no confirmation needed.
	if commandMetadata.GetConnectionId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization for empty connectionId")
	}
	if commandMetadata.GetConnectionId() != d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetConnectionId() {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization for different connectionId: get %v, expected %v", commandMetadata.GetConnectionId(), d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetConnectionId())
	}
	if commandMetadata.GetSequence() <= d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetSequence() {
		return nil, nil
	}
	twinSynchronization := req.GetTwinSynchronization()
	switch twinSynchronization.GetState() {
	case commands.TwinSynchronization_NONE:
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid state(%v)", twinSynchronization.GetState())
	case commands.TwinSynchronization_STARTED:
		if twinSynchronization.GetStartedAt() <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid startedAt(%v)", twinSynchronization.GetStartedAt())
		}
		if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_STARTED {
			if twinSynchronization.GetStartedAt() > d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetStartedAt() {
				return nil, nil
			}
		}
		twinSynchronization.FinishedAt = d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetFinishedAt()
	case commands.TwinSynchronization_FINISHED:
		if twinSynchronization.GetFinishedAt() <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid finishAt(%v)", twinSynchronization.GetStartedAt())
		}
		if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_FINISHED {
			if twinSynchronization.GetFinishedAt() < d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetFinishedAt() {
				return nil, nil
			}
		}
		if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_STARTED {
			if twinSynchronization.GetFinishedAt() < d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetStartedAt() {
				return nil, nil
			}
		}
		twinSynchronization.StartedAt = d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetStartedAt()
	}
	twinSynchronization.CommandMetadata = commandMetadata
	ev := DeviceMetadataUpdated{
		DeviceId:             req.GetDeviceId(),
		Connection:           d.GetDeviceMetadataUpdated().GetConnection(),
		TwinEnabled:          d.GetDeviceMetadataUpdated().GetTwinEnabled(),
		TwinSynchronization:  twinSynchronization,
		AuditContext:         ac,
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
	}
	ok, err := d.HandleDeviceMetadataUpdated(ctx, &ev, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []eventstore.Event{&ev}, nil
}

func (d *DeviceMetadataSnapshotTaken) updateDeviceTwinEnabled(ctx context.Context, req *commands.UpdateDeviceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	ev := DeviceMetadataUpdatePending{
		DeviceId:   req.GetDeviceId(),
		ValidUntil: timeToLive2ValidUntil(req.GetTimeToLive()),
		UpdatePending: &DeviceMetadataUpdatePending_TwinEnabled{
			TwinEnabled: req.GetTwinEnabled(),
		},
		AuditContext:         ac,
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
	}
	err := d.HandleDeviceMetadataUpdatePending(ctx, &ev)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{&ev}, nil
}

func (d *DeviceMetadataSnapshotTaken) updateDeviceMetadata(ctx context.Context, userID string, req *commands.UpdateDeviceMetadataRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())

	_, is_update_twin_enabled := req.GetUpdate().(*commands.UpdateDeviceMetadataRequest_TwinEnabled)
	switch {
	case req.GetConnection() != nil:
		return d.updateDeviceConnection(ctx, req, em, ac)
	case req.GetTwinSynchronization() != nil:
		return d.updateDeviceTwinSynchronization(ctx, req, em, ac)
	case is_update_twin_enabled:
		return d.updateDeviceTwinEnabled(ctx, req, em, ac)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown update type(%T)", req.GetUpdate())
	}
}

func (d *DeviceMetadataSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return nil, err
	}

	switch req := cmd.(type) {
	case *commands.UpdateDeviceMetadataRequest:
		return d.updateDeviceMetadata(ctx, userID, req, newVersion)
	case *commands.ConfirmDeviceMetadataUpdateRequest:
		return d.ConfirmDeviceMetadataUpdate(ctx, userID, req, newVersion, false)
	case *commands.CancelPendingMetadataUpdatesRequest:
		return d.CancelPendingMetadataUpdates(ctx, userID, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (d *DeviceMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &DeviceMetadataSnapshotTaken{
		DeviceId:              d.GetDeviceId(),
		EventMetadata:         MakeEventMeta(d.GetEventMetadata().GetConnectionId(), d.GetEventMetadata().GetSequence(), version),
		DeviceMetadataUpdated: d.GetDeviceMetadataUpdated(),
	}, true
}

func NewDeviceMetadataSnapshotTaken() *DeviceMetadataSnapshotTaken {
	return &DeviceMetadataSnapshotTaken{
		DeviceMetadataUpdated: &DeviceMetadataUpdated{
			TwinEnabled: true,
		},
		EventMetadata: &EventMetadata{},
	}
}
