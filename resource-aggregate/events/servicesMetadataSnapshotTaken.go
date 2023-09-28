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

const (
	eventTypeServicesMetadataSnapshotTaken = "servicesmetadatasnapshottaken"
	writeCostAgainstRead                   = 10
)

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
	online := make(map[string]*ServicesHeartbeat_Heartbeat, len(d.GetServicesMetadataUpdated().GetHeartbeat().GetOnline())+len(upd.GetHeartbeat().GetOnline()))
	for _, v := range d.GetServicesMetadataUpdated().GetHeartbeat().GetOnline() {
		online[v.GetServiceId()] = v
	}
	offline := make(map[string]*ServicesHeartbeat_Heartbeat, len(d.GetServicesMetadataUpdated().GetHeartbeat().GetOffline())+len(upd.GetHeartbeat().GetOffline()))
	for _, v := range d.GetServicesMetadataUpdated().GetHeartbeat().GetOffline() {
		offline[v.GetServiceId()] = v
	}
	// update current state
	for _, v := range upd.GetHeartbeat().GetOnline() {
		online[v.GetServiceId()] = v
		delete(offline, v.GetServiceId())
	}
	for _, v := range upd.GetHeartbeat().GetOffline() {
		offline[v.GetServiceId()] = v
		delete(online, v.GetServiceId())
	}
	// check if there is no service which is online and offline at the same time
	for key := range offline {
		if _, ok := online[key]; ok {
			return false, fmt.Errorf("invalid status: service %v is online and offline", key)
		}
	}
	// update snapshot
	d.ServicesMetadataUpdated = &ServicesMetadataUpdated{
		Heartbeat:            &ServicesHeartbeat{Online: serviceHeartbeatMapToArray(online), Offline: serviceHeartbeatMapToArray(offline)},
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

func serviceHeartbeatMapToArray(m map[string]*ServicesHeartbeat_Heartbeat) []*ServicesHeartbeat_Heartbeat {
	arr := make([]*ServicesHeartbeat_Heartbeat, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	// sort by serviceId and instanceId to make sure that order of services is always the same
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].GetServiceId() < arr[j].GetServiceId()
	})
	return arr
}

func (d *ServicesMetadataSnapshotTaken) updateHeartbeat(ctx context.Context, req *commands.UpdateServiceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	if req.GetHeartbeat().GetServiceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot update status for device %v: invalid instanceId", req.GetHeartbeat().GetServiceId())
	}
	now := time.Now()
	servicesHeartbeat := d.GetServicesMetadataUpdated().GetHeartbeat()
	offline := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetOnline())+len(servicesHeartbeat.GetOffline()))
	for _, v := range servicesHeartbeat.GetOffline() {
		key := v.GetServiceId()
		offline[key] = v
	}

	if offline[req.GetHeartbeat().GetServiceId()] != nil {
		// The service is already offline, and the service needs to be shut down to avoid conflicts in device connection status (ONLINE/OFFLINE).
		return nil, status.Errorf(codes.FailedPrecondition, "cannot update status for device %v: already offline", req.GetHeartbeat().GetServiceId())
	}

	online := make(map[string]*ServicesHeartbeat_Heartbeat, 1)
	for _, v := range servicesHeartbeat.GetOnline() {
		key := v.GetServiceId()
		if v.GetHeartbeatValidUntil() < now.UnixNano() {
			offline[key] = v
		}
	}
	key := req.GetHeartbeat().GetServiceId()

	timeToLive := time.Duration(req.GetHeartbeat().GetTimeToLive())
	// If the request has a valid timestamp, calculate the additional TTL based on processing time.
	if req.GetHeartbeat().GetTimestamp() >= 0 {
		// Calculate the time passed since the request's timestamp and adjust it by a cost factor. In worst case, the timeToLive will be adjusted by 20 minutes.
		processingTime := time.Since(pkgTime.Unix(0, req.GetHeartbeat().GetTimestamp()))
		// Limit the processing time to two minutes, because it will by multiplied by a cost factor.
		if processingTime > time.Minute*2 {
			processingTime = time.Minute * 2
		}
		// If the processing time is positive, add it to the TTL. If it is negative, it means that the service hasn't synced time.
		if processingTime > 0 {
			timeToLive += (processingTime * writeCostAgainstRead)
		}
	}

	online[key] = &ServicesHeartbeat_Heartbeat{
		ServiceId:           req.GetHeartbeat().GetServiceId(),
		HeartbeatValidUntil: now.Add(timeToLive).UnixNano(),
	}
	delete(offline, key)

	ev := ServicesMetadataUpdated{
		Heartbeat:            &ServicesHeartbeat{Online: serviceHeartbeatMapToArray(online), Offline: serviceHeartbeatMapToArray(offline)},
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
	servicesHeartbeat := d.GetServicesMetadataUpdated().GetHeartbeat()
	offline := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetOffline()))
	for _, v := range servicesHeartbeat.GetOffline() {
		key := v.GetServiceId()
		offline[key] = v
	}

	var exist bool
	for _, v := range req.Heartbeat {
		key := v.GetServiceId()
		if _, ok := offline[key]; ok {
			delete(offline, key)
			exist = true
		}
	}
	if !exist {
		return nil, nil
	}
	// take snapshot to dump full state of services
	d.ServicesMetadataUpdated = &ServicesMetadataUpdated{
		Heartbeat:            &ServicesHeartbeat{Online: d.GetServicesMetadataUpdated().GetHeartbeat().GetOnline(), Offline: serviceHeartbeatMapToArray(offline)},
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
	case req.GetHeartbeat() != nil:
		return d.updateHeartbeat(ctx, req, em, ac)
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
	Heartbeat []*ServicesHeartbeat_Heartbeat
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
