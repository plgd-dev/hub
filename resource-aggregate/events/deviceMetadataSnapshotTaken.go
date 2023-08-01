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
	return commands.MakeStatusResourceUUID(d.GetDeviceId()).String()
}

func (d *DeviceMetadataSnapshotTaken) GroupID() string {
	return d.GetDeviceId()
}

func (d *DeviceMetadataSnapshotTaken) IsSnapshot() bool {
	return true
}

func (d *DeviceMetadataSnapshotTaken) ETag() *eventstore.ETagData {
	return nil
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

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdated(_ context.Context, upd *DeviceMetadataUpdated, confirm bool) (bool, error) {
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

	d.DeviceId = upd.GetDeviceId()
	d.DeviceMetadataUpdated = upd
	d.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataSnapshotTaken(_ context.Context, s *DeviceMetadataSnapshotTaken) {
	d.CopyData(s)
}

func (d *DeviceMetadataSnapshotTaken) HandleDeviceMetadataUpdatePending(_ context.Context, updatePending *DeviceMetadataUpdatePending) error {
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

func (d *DeviceMetadataSnapshotTaken) cancelDeviceMetadataUpdate(ctx context.Context, req *commands.ConfirmDeviceMetadataUpdateRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
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
}

func (d *DeviceMetadataSnapshotTaken) updateTwinEnabled(ctx context.Context, req *commands.ConfirmDeviceMetadataUpdateRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	twinSynchronization := d.GetDeviceMetadataUpdated().GetTwinSynchronization()
	if twinSynchronization == nil {
		twinSynchronization = &commands.TwinSynchronization{
			State:           commands.TwinSynchronization_OUT_OF_SYNC,
			CommandMetadata: req.GetCommandMetadata(),
		}
	}
	if req.GetCommandMetadata().GetConnectionId() == d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetConnectionId() {
		switch req.GetTwinEnabled() {
		case true:
			if twinSynchronization.GetState() == commands.TwinSynchronization_DISABLED {
				twinSynchronization.State = commands.TwinSynchronization_OUT_OF_SYNC
			}
		case false:
			twinSynchronization.State = commands.TwinSynchronization_DISABLED
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
}

func (d *DeviceMetadataSnapshotTaken) updateTwinForceResynchronization(ctx context.Context, req *commands.ConfirmDeviceMetadataUpdateRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	if !req.GetTwinForceResynchronization() {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid forceResynchronization(%v)", req.GetTwinForceResynchronization())
	}
	twinSynchronization := d.GetDeviceMetadataUpdated().GetTwinSynchronization()
	if twinSynchronization == nil {
		twinSynchronization = &commands.TwinSynchronization{
			CommandMetadata: req.GetCommandMetadata(),
		}
	}
	twinSynchronization.State = commands.TwinSynchronization_OUT_OF_SYNC
	twinSynchronization.ForceResynchronizationAt = em.GetTimestamp()
	ev := DeviceMetadataUpdated{
		DeviceId:             req.GetDeviceId(),
		Connection:           d.GetDeviceMetadataUpdated().GetConnection(),
		TwinEnabled:          true,
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
}

func (d *DeviceMetadataSnapshotTaken) ConfirmDeviceMetadataUpdate(ctx context.Context, userID, hubID string, req *commands.ConfirmDeviceMetadataUpdateRequest, newVersion uint64, cancel bool) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, hubID)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())
	_, is_confirm_twin_enabled := req.GetConfirm().(*commands.ConfirmDeviceMetadataUpdateRequest_TwinEnabled)
	_, is_confirm_twin_force_resynchronization := req.GetConfirm().(*commands.ConfirmDeviceMetadataUpdateRequest_TwinForceResynchronization)
	switch {
	case cancel:
		return d.cancelDeviceMetadataUpdate(ctx, req, em, ac)
	case is_confirm_twin_enabled:
		return d.updateTwinEnabled(ctx, req, em, ac)
	case is_confirm_twin_force_resynchronization:
		return d.updateTwinForceResynchronization(ctx, req, em, ac)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown confirm type(%T)", req.GetConfirm())
	}
}

func (d *DeviceMetadataSnapshotTaken) CancelPendingMetadataUpdates(ctx context.Context, userID, hubID string, req *commands.CancelPendingMetadataUpdatesRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}
	correlationIdFilter := strings.MakeSet(req.GetCorrelationIdFilter()...)
	events := make([]eventstore.Event, 0, 4)
	for _, event := range d.GetUpdatePendings() {
		if len(correlationIdFilter) != 0 && !correlationIdFilter.HasOneOf(event.GetAuditContext().GetCorrelationId()) {
			continue
		}
		ev, err := d.ConfirmDeviceMetadataUpdate(ctx, userID, hubID, &commands.ConfirmDeviceMetadataUpdateRequest{
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

func (d *DeviceMetadataSnapshotTaken) prepareTwinSynchronization() *commands.TwinSynchronization {
	twinSynchronization := d.GetDeviceMetadataUpdated().GetTwinSynchronization()
	// check if it is new connection
	// reset twinSynchronization
	if twinSynchronization == nil {
		twinSynchronization = &commands.TwinSynchronization{}
	}
	return twinSynchronization
}

func (d *DeviceMetadataSnapshotTaken) getTwinSynchronizationForConnectedDevice(req *commands.UpdateDeviceMetadataRequest) (*commands.TwinSynchronization, error) {
	twinSynchronization := d.prepareTwinSynchronization()
	if req.GetConnection().GetId() != d.GetDeviceMetadataUpdated().GetConnection().GetId() || !d.GetDeviceMetadataUpdated().GetConnection().IsOnline() {
		if d.GetDeviceMetadataUpdated().GetTwinEnabled() {
			twinSynchronization.State = commands.TwinSynchronization_OUT_OF_SYNC
		} else {
			twinSynchronization.State = commands.TwinSynchronization_DISABLED
		}
		twinSynchronization.CommandMetadata = req.GetCommandMetadata()
	}
	return twinSynchronization, nil
}

func (d *DeviceMetadataSnapshotTaken) getTwinSynchronizationForDisconnectedDevice(req *commands.UpdateDeviceMetadataRequest) (*commands.TwinSynchronization, error) {
	twinSynchronization := d.prepareTwinSynchronization()
	if d.DeviceMetadataUpdated.GetConnection().IsOnline() && !req.GetConnection().IsOnline() && d.DeviceMetadataUpdated.GetConnection().GetId() != req.GetConnection().GetId() {
		// if previous status was online and new status is offline, the connectionId must be the same
		return nil, status.Errorf(codes.InvalidArgument, "cannot update connection status online(id='%v') to offline(id='%v'): connectionId mismatch", d.DeviceMetadataUpdated.GetConnection().GetId(), req.GetConnection().GetId())
	}
	if d.GetDeviceMetadataUpdated().GetTwinEnabled() {
		twinSynchronization.State = commands.TwinSynchronization_OUT_OF_SYNC
	} else {
		twinSynchronization.State = commands.TwinSynchronization_DISABLED
	}
	twinSynchronization.CommandMetadata = req.GetCommandMetadata()
	return twinSynchronization, nil
}

func (d *DeviceMetadataSnapshotTaken) updateDeviceConnection(ctx context.Context, req *commands.UpdateDeviceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	// it is expected that the device updates the status on its own. no confirmation needed.
	req.GetConnection().Id = req.GetCommandMetadata().GetConnectionId()
	if req.GetConnection().GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update connection status for empty connectionId")
	}
	if !req.GetConnection().IsOnline() {
		if em.GetVersion() == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot update connection status to offline for not existing device %v", req.GetDeviceId())
		}
		// only online status can update protocol
		req.GetConnection().Protocol = d.GetDeviceMetadataUpdated().GetConnection().GetProtocol()
	}

	// keep last connected at from the previous event
	lastConnectedAt := d.GetDeviceMetadataUpdated().GetConnection().GetConnectedAt()
	if req.GetConnection().GetConnectedAt() < lastConnectedAt {
		req.GetConnection().ConnectedAt = lastConnectedAt
	}

	getTwinSynchronization := d.getTwinSynchronizationForDisconnectedDevice
	if req.GetConnection().IsOnline() {
		getTwinSynchronization = d.getTwinSynchronizationForConnectedDevice
	}
	twinSynchronization, err := getTwinSynchronization(req)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []eventstore.Event{&ev}, nil
}

func (d *DeviceMetadataSnapshotTaken) prepareTwinSynchronizationToSyncing(twinSynchronization *commands.TwinSynchronization) (bool, error) {
	if !d.GetDeviceMetadataUpdated().GetTwinEnabled() {
		return false, status.Errorf(codes.InvalidArgument, "cannot update twinSynchronization to %v: twin is disabled", commands.TwinSynchronization_SYNCING)
	}
	if twinSynchronization.GetSyncingAt() <= 0 {
		return false, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid startedAt(%v)", twinSynchronization.GetSyncingAt())
	}
	if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_SYNCING {
		if twinSynchronization.GetSyncingAt() > d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetSyncingAt() {
			return false, nil
		}
	}
	twinSynchronization.InSyncAt = d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetInSyncAt()
	return true, nil
}

func (d *DeviceMetadataSnapshotTaken) prepareTwinSynchronizationToInSync(twinSynchronization *commands.TwinSynchronization) (bool, error) {
	if !d.GetDeviceMetadataUpdated().GetTwinEnabled() {
		return false, status.Errorf(codes.InvalidArgument, "cannot update twinSynchronization to %v: twin is disabled", commands.TwinSynchronization_IN_SYNC)
	}
	if twinSynchronization.GetInSyncAt() <= 0 {
		return false, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid finishAt(%v)", twinSynchronization.GetSyncingAt())
	}
	if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC {
		if twinSynchronization.GetInSyncAt() < d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetInSyncAt() {
			return false, nil
		}
	}
	if d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_SYNCING {
		if twinSynchronization.GetInSyncAt() < d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetSyncingAt() {
			return false, nil
		}
	}
	twinSynchronization.SyncingAt = d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetSyncingAt()
	return true, nil
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
	if em.GetVersion() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization for not existing device %v", req.GetDeviceId())
	}
	if commandMetadata.GetSequence() <= d.GetDeviceMetadataUpdated().GetTwinSynchronization().GetCommandMetadata().GetSequence() {
		return nil, nil
	}
	twinSynchronization := req.GetTwinSynchronization()
	switch twinSynchronization.GetState() {
	case commands.TwinSynchronization_OUT_OF_SYNC:
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin synchronization with invalid state(%v)", twinSynchronization.GetState())
	case commands.TwinSynchronization_SYNCING:
		if ok, err := d.prepareTwinSynchronizationToSyncing(twinSynchronization); err != nil || !ok {
			return nil, err
		}
	case commands.TwinSynchronization_IN_SYNC:
		if ok, err := d.prepareTwinSynchronizationToInSync(twinSynchronization); err != nil || !ok {
			return nil, err
		}
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
	if em.GetVersion() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin enabled for not existing device %v", req.GetDeviceId())
	}
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

func (d *DeviceMetadataSnapshotTaken) updateDeviceTwinForceResynchronization(ctx context.Context, req *commands.UpdateDeviceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	if em.GetVersion() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin enabled for not existing device %v", req.GetDeviceId())
	}
	if !req.GetTwinForceResynchronization() {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update twin force resynchronization with invalid forceResynchronization(%v)", req.GetTwinForceResynchronization())
	}
	ev := DeviceMetadataUpdatePending{
		DeviceId:   req.GetDeviceId(),
		ValidUntil: timeToLive2ValidUntil(req.GetTimeToLive()),
		UpdatePending: &DeviceMetadataUpdatePending_TwinForceResynchronization{
			TwinForceResynchronization: req.GetTwinForceResynchronization(),
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

func (d *DeviceMetadataSnapshotTaken) updateDeviceMetadata(ctx context.Context, userID, hubID string, req *commands.UpdateDeviceMetadataRequest, newVersion uint64) ([]eventstore.Event, error) {
	if req.GetCommandMetadata() == nil {
		return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
	}

	em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, hubID)
	ac := commands.NewAuditContext(userID, req.GetCorrelationId())

	_, is_update_twin_enabled := req.GetUpdate().(*commands.UpdateDeviceMetadataRequest_TwinEnabled)
	_, is_process_twin_force_resynchronization := req.GetUpdate().(*commands.UpdateDeviceMetadataRequest_TwinForceResynchronization)
	switch {
	case req.GetConnection() != nil:
		return d.updateDeviceConnection(ctx, req, em, ac)
	case req.GetTwinSynchronization() != nil:
		return d.updateDeviceTwinSynchronization(ctx, req, em, ac)
	case is_update_twin_enabled:
		return d.updateDeviceTwinEnabled(ctx, req, em, ac)
	case is_process_twin_force_resynchronization:
		return d.updateDeviceTwinForceResynchronization(ctx, req, em, ac)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown update type(%T)", req.GetUpdate())
	}
}

func (d *DeviceMetadataSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return nil, err
	}
	hubID := HubIDFromCtx(ctx)
	if hubID == "" {
		return nil, fmt.Errorf("hubID not found")
	}

	switch req := cmd.(type) {
	case *commands.UpdateDeviceMetadataRequest:
		return d.updateDeviceMetadata(ctx, userID, hubID, req, newVersion)
	case *commands.ConfirmDeviceMetadataUpdateRequest:
		return d.ConfirmDeviceMetadataUpdate(ctx, userID, hubID, req, newVersion, false)
	case *commands.CancelPendingMetadataUpdatesRequest:
		return d.CancelPendingMetadataUpdates(ctx, userID, hubID, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (d *DeviceMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &DeviceMetadataSnapshotTaken{
		DeviceId:              d.GetDeviceId(),
		EventMetadata:         MakeEventMeta(d.GetEventMetadata().GetConnectionId(), d.GetEventMetadata().GetSequence(), version, d.GetEventMetadata().GetHubId()),
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
