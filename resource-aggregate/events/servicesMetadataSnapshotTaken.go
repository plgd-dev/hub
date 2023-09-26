package events

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeServicesMetadataSnapshotTaken = "servicesmetadatasnapshottaken"

func (d *ServicesMetadataSnapshotTaken) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *ServicesMetadataSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *ServicesMetadataSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *ServicesMetadataSnapshotTaken) EventType() string {
	return eventTypeServicesMetadataSnapshotTaken
}

func (d *ServicesMetadataSnapshotTaken) AggregateID() string {
	return commands.MakeServicesResourceUUID(d.GetEventMetadata().GetHubId()).String()
}

func (d *ServicesMetadataSnapshotTaken) GroupID() string {
	return d.GetEventMetadata().GetHubId()
}

func (d *ServicesMetadataSnapshotTaken) IsSnapshot() bool {
	return true
}

func (d *ServicesMetadataSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *ServicesMetadataSnapshotTaken) ETag() *eventstore.ETagData {
	return nil
}

func (d *ServicesMetadataSnapshotTaken) ServiceID() (string, bool) {
	return "", false
}

func (d *ServicesMetadataSnapshotTaken) CopyData(event *ServicesMetadataSnapshotTaken) {
	d.ServicesMetadataUpdated = event.GetServicesMetadataUpdated()
	d.EventMetadata = event.GetEventMetadata()
}

func (d *ServicesMetadataSnapshotTaken) CheckInitialized() bool {
	return d.GetServicesMetadataUpdated() != nil &&
		d.GetEventMetadata() != nil
}

func (d *ServicesMetadataSnapshotTaken) HandleServicesMetadataUpdated(_ context.Context, upd *ServicesMetadataUpdated) (bool, error) {
	if d.ServicesMetadataUpdated.Equal(upd) {
		return false, nil
	}
	online := make(map[string]*ServicesStatus_Status, len(d.GetServicesMetadataUpdated().GetStatus().GetOnline())+len(upd.GetStatus().GetOnline()))
	for _, v := range d.GetServicesMetadataUpdated().GetStatus().GetOnline() {
		online[v.GetId()] = v
	}
	offline := make(map[string]*ServicesStatus_Status, len(d.GetServicesMetadataUpdated().GetStatus().GetOffline())+len(upd.GetStatus().GetOffline()))
	for _, v := range d.GetServicesMetadataUpdated().GetStatus().GetOffline() {
		offline[v.GetId()] = v
	}
	// update current state
	for _, v := range upd.GetStatus().GetOnline() {
		online[v.GetId()] = v
		delete(offline, v.GetId())
	}
	for _, v := range upd.GetStatus().GetOffline() {
		offline[v.GetId()] = v
		delete(online, v.GetId())
	}
	// check if there is no service which is online and offline at the same time
	for key := range online {
		if _, ok := offline[key]; ok {
			return false, fmt.Errorf("invalid status: service %v is online and offline", key)
		}
	}
	for key := range offline {
		if _, ok := online[key]; ok {
			return false, fmt.Errorf("invalid status: service %v is online and offline", key)
		}
	}
	// update snapshot
	d.ServicesMetadataUpdated = &ServicesMetadataUpdated{
		Status:               &ServicesStatus{Online: serviceStatusMapToArray(online), Offline: serviceStatusMapToArray(offline)},
		EventMetadata:        upd.GetEventMetadata(),
		OpenTelemetryCarrier: upd.GetOpenTelemetryCarrier(),
	}
	d.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (d *ServicesMetadataSnapshotTaken) HandleServicesMetadataSnapshotTaken(_ context.Context, s *ServicesMetadataSnapshotTaken) {
	d.CopyData(s)
}

func (d *ServicesMetadataSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		switch eu.EventType() {
		case (&ServicesMetadataSnapshotTaken{}).EventType():
			var s ServicesMetadataSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			d.HandleServicesMetadataSnapshotTaken(ctx, &s)
		case (&ServicesMetadataUpdated{}).EventType():
			var s ServicesMetadataUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_, _ = d.HandleServicesMetadataUpdated(ctx, &s)
		}
	}
	return iter.Err()
}

func serviceStatusMapToArray(m map[string]*ServicesStatus_Status) []*ServicesStatus_Status {
	arr := make([]*ServicesStatus_Status, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	// sort by serviceId and instanceId to make sure that order of services is always the same
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].GetId() < arr[j].GetId()
	})
	return arr
}

func (d *ServicesMetadataSnapshotTaken) updateStatus(ctx context.Context, req *commands.UpdateServiceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	if req.GetStatus().GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update status for device %v: invalid instanceId", req.GetStatus().GetId())
	}
	now := time.Now()
	servicesStatus := d.GetServicesMetadataUpdated().GetStatus()
	offline := make(map[string]*ServicesStatus_Status, len(servicesStatus.GetOnline())+len(servicesStatus.GetOffline()))
	for _, v := range servicesStatus.GetOffline() {
		key := v.GetId()
		offline[key] = v
	}

	if offline[req.GetStatus().GetId()] != nil {
		// The service is already offline, and the service needs to be shut down to avoid conflicts in device connection status (ONLINE/OFFLINE).
		return nil, status.Errorf(codes.FailedPrecondition, "cannot update status for device %v: already offline", req.GetStatus().GetId())
	}

	online := make(map[string]*ServicesStatus_Status, 1)
	for _, v := range servicesStatus.GetOnline() {
		key := v.GetId()
		if v.GetOnlineValidUntil() < now.UnixNano() {
			offline[key] = v
		}
	}
	key := req.GetStatus().GetId()
	online[key] = &ServicesStatus_Status{
		Id:               req.GetStatus().GetId(),
		OnlineValidUntil: now.Add(time.Duration(req.GetStatus().GetTimeToLive())).UnixNano(),
	}
	delete(offline, key)

	ev := ServicesMetadataUpdated{
		Status:               &ServicesStatus{Online: serviceStatusMapToArray(online), Offline: serviceStatusMapToArray(offline)},
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		AuditContext:         ac,
	}
	ok, err := d.HandleServicesMetadataUpdated(ctx, &ev)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []eventstore.Event{&ev}, nil
}

// return snapshot as event if any service was confirmed offline
func (d *ServicesMetadataSnapshotTaken) confirmOfflineServices(ctx context.Context, req *ConfirmOfflineServicesRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	servicesStatus := d.GetServicesMetadataUpdated().GetStatus()
	offline := make(map[string]*ServicesStatus_Status, len(servicesStatus.GetOffline()))
	for _, v := range servicesStatus.GetOffline() {
		key := v.GetId()
		offline[key] = v
	}

	removed := make([]*ServicesStatus_Status, 0, len(req.Status))
	for _, v := range req.Status {
		key := v.GetId()
		if _, ok := offline[key]; ok {
			delete(offline, key)
			removed = append(removed, v)
		}
	}
	if len(removed) == 0 {
		return nil, nil
	}
	// take snapshot to dump full state of services
	d.ServicesMetadataUpdated = &ServicesMetadataUpdated{
		Status:               &ServicesStatus{Online: d.GetServicesMetadataUpdated().GetStatus().GetOnline(), Offline: serviceStatusMapToArray(offline)},
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		AuditContext:         ac,
	}
	d.EventMetadata = em
	snapshot, ok := d.TakeSnapshot(em.GetVersion())
	if !ok {
		return nil, fmt.Errorf("cannot take snapshot")
	}
	return []eventstore.Event{snapshot}, nil
}

func (d *ServicesMetadataSnapshotTakenForCommand) updateServiceMetadataRequest(ctx context.Context, req *commands.UpdateServiceMetadataRequest, newVersion uint64) ([]eventstore.Event, error) {
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	em := MakeEventMeta(connectionID, 0, newVersion, d.hubID)
	ac := commands.NewAuditContext(d.userID, "", d.owner)

	switch {
	case req.GetStatus() != nil:
		return d.updateStatus(ctx, req, em, ac)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown update type(%T)", req.GetUpdate())
	}
}

func (d *ServicesMetadataSnapshotTakenForCommand) confirmOfflineServicesRequest(ctx context.Context, req *ConfirmOfflineServicesRequest, newVersion uint64) ([]eventstore.Event, error) {
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	em := MakeEventMeta(connectionID, 0, newVersion, d.hubID)
	ac := commands.NewAuditContext(d.userID, "", d.owner)

	return d.confirmOfflineServices(ctx, req, em, ac)
}

type ConfirmOfflineServicesRequest struct {
	Status []*ServicesStatus_Status
}

func (d *ServicesMetadataSnapshotTakenForCommand) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	switch req := cmd.(type) {
	case *commands.UpdateServiceMetadataRequest:
		return d.updateServiceMetadataRequest(ctx, req, newVersion)
	case *ConfirmOfflineServicesRequest:
		return d.confirmOfflineServicesRequest(ctx, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (d *ServicesMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &ServicesMetadataSnapshotTaken{
		EventMetadata:           MakeEventMeta(d.GetEventMetadata().GetConnectionId(), d.GetEventMetadata().GetSequence(), version, d.GetEventMetadata().GetHubId()),
		ServicesMetadataUpdated: d.GetServicesMetadataUpdated(),
	}, true
}

type ServicesMetadataSnapshotTakenForCommand struct {
	userID string
	owner  string
	hubID  string
	*ServicesMetadataSnapshotTaken
}

func NewServicesMetadataSnapshotTakenForCommand(userID string, owner string, hubID string) *ServicesMetadataSnapshotTakenForCommand {
	return &ServicesMetadataSnapshotTakenForCommand{
		ServicesMetadataSnapshotTaken: NewServicesMetadataSnapshotTaken(),
		userID:                        userID,
		owner:                         owner,
		hubID:                         hubID,
	}
}

func NewServicesMetadataSnapshotTaken() *ServicesMetadataSnapshotTaken {
	return &ServicesMetadataSnapshotTaken{
		ServicesMetadataUpdated: &ServicesMetadataUpdated{},
		EventMetadata:           &EventMetadata{},
	}
}
