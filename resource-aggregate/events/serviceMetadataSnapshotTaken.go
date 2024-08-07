package events

import (
	"context"
	"errors"
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
	eventTypeServiceMetadataSnapshotTaken = "ServiceMetadataSnapshotTaken"
	writeCostAgainstRead                  = 10
)

func (d *ServiceMetadataSnapshotTaken) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *ServiceMetadataSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *ServiceMetadataSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *ServiceMetadataSnapshotTaken) EventType() string {
	return eventTypeServiceMetadataSnapshotTaken
}

func (d *ServiceMetadataSnapshotTaken) AggregateID() string {
	return commands.MakeServicesResourceUUID(d.GetEventMetadata().GetHubId()).String()
}

func (d *ServiceMetadataSnapshotTaken) GroupID() string {
	return d.GetEventMetadata().GetHubId()
}

func (d *ServiceMetadataSnapshotTaken) IsSnapshot() bool {
	return true
}

func (d *ServiceMetadataSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *ServiceMetadataSnapshotTaken) ETag() *eventstore.ETagData {
	return nil
}

func (d *ServiceMetadataSnapshotTaken) ServiceID() (string, bool) {
	return "", false
}

func (d *ServiceMetadataSnapshotTaken) Types() []string {
	return nil
}

func (d *ServiceMetadataSnapshotTaken) CopyData(event *ServiceMetadataSnapshotTaken) {
	d.ServiceMetadataUpdated = event.GetServiceMetadataUpdated()
	d.EventMetadata = event.GetEventMetadata()
}

func (d *ServiceMetadataSnapshotTaken) CheckInitialized() bool {
	return d.GetServiceMetadataUpdated() != nil &&
		d.GetEventMetadata() != nil
}

func (d *ServiceMetadataSnapshotTaken) HandleServiceMetadataUpdated(_ context.Context, upd *ServiceMetadataUpdated) (bool, error) {
	if d.GetServiceMetadataUpdated().Equal(upd) {
		return false, nil
	}
	valid := make(map[string]*ServicesHeartbeat_Heartbeat, len(d.GetServiceMetadataUpdated().GetServicesHeartbeat().GetValid())+len(upd.GetServicesHeartbeat().GetValid()))
	for _, v := range d.GetServiceMetadataUpdated().GetServicesHeartbeat().GetValid() {
		valid[v.GetServiceId()] = v
	}
	expired := make(map[string]*ServicesHeartbeat_Heartbeat, len(d.GetServiceMetadataUpdated().GetServicesHeartbeat().GetExpired())+len(upd.GetServicesHeartbeat().GetExpired()))
	for _, v := range d.GetServiceMetadataUpdated().GetServicesHeartbeat().GetExpired() {
		expired[v.GetServiceId()] = v
	}
	// update current state
	for _, v := range upd.GetServicesHeartbeat().GetValid() {
		valid[v.GetServiceId()] = v
		delete(expired, v.GetServiceId())
	}
	for _, v := range upd.GetServicesHeartbeat().GetExpired() {
		expired[v.GetServiceId()] = v
		delete(valid, v.GetServiceId())
	}
	// check if there is no service which is valid and expired at the same time
	for key := range expired {
		if _, ok := valid[key]; ok {
			return false, fmt.Errorf("invalid status: service %v is valid and expired", key)
		}
	}
	// update snapshot
	d.ServiceMetadataUpdated = &ServiceMetadataUpdated{
		ServicesHeartbeat:    &ServicesHeartbeat{Valid: serviceHeartbeatMapToArray(valid), Expired: serviceHeartbeatMapToArray(expired)},
		EventMetadata:        upd.GetEventMetadata(),
		OpenTelemetryCarrier: upd.GetOpenTelemetryCarrier(),
	}
	d.EventMetadata = upd.GetEventMetadata()
	return true, nil
}

func (d *ServiceMetadataSnapshotTaken) HandleServiceMetadataSnapshotTaken(_ context.Context, s *ServiceMetadataSnapshotTaken) {
	d.CopyData(s)
}

func (d *ServiceMetadataSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		switch eu.EventType() {
		case (&ServiceMetadataSnapshotTaken{}).EventType():
			var s ServiceMetadataSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			d.HandleServiceMetadataSnapshotTaken(ctx, &s)
		case (&ServiceMetadataUpdated{}).EventType():
			var s ServiceMetadataUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			_, _ = d.HandleServiceMetadataUpdated(ctx, &s)
		}
	}
	return iter.Err()
}

func (h *ServicesHeartbeat_Heartbeat) CopyData(h1 *ServicesHeartbeat_Heartbeat) {
	h.ServiceId = h1.GetServiceId()
	h.ValidUntil = h1.GetValidUntil()
}

func serviceHeartbeatMapToArray(m map[string]*ServicesHeartbeat_Heartbeat) []*ServicesHeartbeat_Heartbeat {
	arr := make([]*ServicesHeartbeat_Heartbeat, 0, len(m))
	for _, v := range m {
		var h ServicesHeartbeat_Heartbeat
		h.CopyData(v)
		arr = append(arr, &h)
	}
	// sort by serviceId and instanceId to make sure that order of services is always the same
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].GetServiceId() < arr[j].GetServiceId()
	})
	return arr
}

func calculateTimeToLive(now time.Time, req *commands.UpdateServiceMetadataRequest) time.Duration {
	timeToLive := time.Duration(req.GetHeartbeat().GetTimeToLive())
	// If the request has a valid timestamp, calculate the additional TTL based on processing time.
	if req.GetHeartbeat().GetTimestamp() != 0 {
		timestamp := pkgTime.Unix(0, req.GetHeartbeat().GetTimestamp())
		if timestamp.IsZero() || now.Before(timestamp) {
			// If the timestamp is invalid or in the future, return the TTL as is.
			return timeToLive
		}
		// Calculate the time passed since the request's timestamp and adjust it by a cost factor. In worst case, the timeToLive will be adjusted by 20 minutes.
		processingTime := time.Since(timestamp)
		// Limit the processing time to two minutes, because it will by multiplied by a cost factor.
		if processingTime > time.Minute*2 {
			processingTime = time.Minute * 2
		}
		// If the processing time is positive, add it to the TTL. If it is negative, it means that the service hasn't synced time.
		if processingTime > 0 {
			timeToLive += (processingTime * writeCostAgainstRead)
		}
	}
	return timeToLive
}

func (d *ServiceMetadataSnapshotTaken) prepareHeartbeatForService(req *commands.UpdateServiceMetadataRequest, now time.Time) (timeToLive time.Duration, expiredServices []*ServicesHeartbeat_Heartbeat, err error) {
	errActionStr := "update"
	if req.GetHeartbeat().GetRegister() {
		errActionStr = "register"
	}
	if req.GetHeartbeat().GetServiceId() == "" {
		return 0, nil, status.Errorf(codes.InvalidArgument, "cannot %v heartbeat for service %v: invalid serviceId", errActionStr, req.GetHeartbeat().GetServiceId())
	}
	servicesHeartbeat := d.GetServiceMetadataUpdated().GetServicesHeartbeat()
	expired := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetValid())+len(servicesHeartbeat.GetExpired()))
	for _, v := range servicesHeartbeat.GetExpired() {
		expired[v.GetServiceId()] = v
	}

	if expired[req.GetHeartbeat().GetServiceId()] != nil {
		// The service is already expired, and the service needs to be shut down to avoid conflicts in device connection status (ONLINE/OFFLINE).
		return 0, nil, status.Errorf(codes.FailedPrecondition, "cannot %v heartbeat for service %v: already expired", errActionStr, req.GetHeartbeat().GetServiceId())
	}

	valid := make(map[string]*ServicesHeartbeat_Heartbeat, 1)
	newExpired := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetValid())+len(servicesHeartbeat.GetExpired()))
	for _, v := range servicesHeartbeat.GetValid() {
		validUntil := pkgTime.Unix(0, v.GetValidUntil())
		if validUntil.Before(now) {
			expired[v.GetServiceId()] = v
			newExpired[v.GetServiceId()] = v
		} else {
			valid[v.GetServiceId()] = v
		}
	}

	// register == true - only insert is allowed
	if req.GetHeartbeat().GetRegister() && (valid[req.GetHeartbeat().GetServiceId()] != nil || expired[req.GetHeartbeat().GetServiceId()] != nil) {
		// The service is already registered, and the service needs to be shut down to avoid conflicts in device connection status (ONLINE/OFFLINE).
		return 0, nil, status.Errorf(codes.FailedPrecondition, "cannot %v heartbeat for service %v: already registered", errActionStr, req.GetHeartbeat().GetServiceId())
	}

	// register == false - only update is allowed
	if !req.GetHeartbeat().GetRegister() && (valid[req.GetHeartbeat().GetServiceId()] == nil && expired[req.GetHeartbeat().GetServiceId()] == nil) {
		// The service is not registered, and the service needs to be shut down to avoid conflicts in device connection status (ONLINE/OFFLINE).
		return 0, nil, status.Errorf(codes.FailedPrecondition, "cannot %v heartbeat for service %v: not registered", errActionStr, req.GetHeartbeat().GetServiceId())
	}

	delete(newExpired, req.GetHeartbeat().GetServiceId())
	return calculateTimeToLive(now, req), serviceHeartbeatMapToArray(newExpired), nil
}

func (d *ServiceMetadataSnapshotTaken) updateHeartbeatForService(ctx context.Context, req *commands.UpdateServiceMetadataRequest, now time.Time, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	timeToLive, newExpired, err := d.prepareHeartbeatForService(req, now)
	if err != nil {
		return nil, err
	}

	ev := ServiceMetadataUpdated{
		ServicesHeartbeat: &ServicesHeartbeat{Valid: []*ServicesHeartbeat_Heartbeat{
			{
				ServiceId:  req.GetHeartbeat().GetServiceId(),
				ValidUntil: now.Add(timeToLive).UnixNano(),
			},
		}, Expired: newExpired},
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		AuditContext:         ac,
	}
	ok, err := d.HandleServiceMetadataUpdated(ctx, &ev)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []eventstore.Event{&ev}, nil
}

func (d *ServiceMetadataSnapshotTaken) updateHeartbeatCheckExpiredServices(ctx context.Context, now time.Time, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	servicesHeartbeat := d.GetServiceMetadataUpdated().GetServicesHeartbeat()
	expired := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetValid()))
	for _, v := range servicesHeartbeat.GetValid() {
		validUntil := pkgTime.Unix(0, v.GetValidUntil())
		if validUntil.Before(now) {
			expired[v.GetServiceId()] = v
		}
	}
	if len(expired) == 0 {
		return nil, nil
	}
	ev := ServiceMetadataUpdated{
		ServicesHeartbeat:    &ServicesHeartbeat{Valid: nil, Expired: serviceHeartbeatMapToArray(expired)},
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		AuditContext:         ac,
	}
	ok, err := d.HandleServiceMetadataUpdated(ctx, &ev)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []eventstore.Event{&ev}, nil
}

func (d *ServiceMetadataSnapshotTaken) updateHeartbeat(ctx context.Context, req *commands.UpdateServiceMetadataRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	now := time.Now()
	if req.GetHeartbeat().GetServiceId() == "" && req.GetHeartbeat().GetTimeToLive() == 0 {
		return d.updateHeartbeatCheckExpiredServices(ctx, now, em, ac)
	}
	return d.updateHeartbeatForService(ctx, req, now, em, ac)
}

// return snapshot as event if any service was confirmed expired
func (d *ServiceMetadataSnapshotTaken) confirmExpiredServices(ctx context.Context, req *ConfirmExpiredServicesRequest, em *EventMetadata, ac *commands.AuditContext) ([]eventstore.Event, error) {
	servicesHeartbeat := d.GetServiceMetadataUpdated().GetServicesHeartbeat()
	expired := make(map[string]*ServicesHeartbeat_Heartbeat, len(servicesHeartbeat.GetExpired()))
	for _, v := range servicesHeartbeat.GetExpired() {
		key := v.GetServiceId()
		expired[key] = v
	}

	var exist bool
	for _, v := range req.Heartbeat {
		key := v.GetServiceId()
		if _, ok := expired[key]; ok {
			delete(expired, key)
			exist = true
		}
	}
	if !exist {
		return nil, nil
	}
	// take snapshot to dump full state of services
	d.ServiceMetadataUpdated = &ServiceMetadataUpdated{
		ServicesHeartbeat:    &ServicesHeartbeat{Valid: d.GetServiceMetadataUpdated().GetServicesHeartbeat().GetValid(), Expired: serviceHeartbeatMapToArray(expired)},
		EventMetadata:        em,
		OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		AuditContext:         ac,
	}
	d.EventMetadata = em
	snapshot, ok := d.TakeSnapshot(em.GetVersion())
	if !ok {
		return nil, errors.New("cannot take snapshot")
	}
	return []eventstore.Event{snapshot}, nil
}

func (d *ServiceMetadataSnapshotTakenForCommand) updateServiceMetadataRequest(ctx context.Context, req *commands.UpdateServiceMetadataRequest, newVersion uint64) ([]eventstore.Event, error) {
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

func (d *ServiceMetadataSnapshotTakenForCommand) confirmExpiredServicesRequest(ctx context.Context, req *ConfirmExpiredServicesRequest, newVersion uint64) ([]eventstore.Event, error) {
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	em := MakeEventMeta(connectionID, 0, newVersion, d.hubID)
	ac := commands.NewAuditContext(d.userID, "", d.owner)

	return d.confirmExpiredServices(ctx, req, em, ac)
}

type ConfirmExpiredServicesRequest struct {
	Heartbeat []*ServicesHeartbeat_Heartbeat
}

func (d *ServiceMetadataSnapshotTakenForCommand) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	switch req := cmd.(type) {
	case *commands.UpdateServiceMetadataRequest:
		return d.updateServiceMetadataRequest(ctx, req, newVersion)
	case *ConfirmExpiredServicesRequest:
		return d.confirmExpiredServicesRequest(ctx, req, newVersion)
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (d *ServiceMetadataSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	return &ServiceMetadataSnapshotTaken{
		EventMetadata:          MakeEventMeta(d.GetEventMetadata().GetConnectionId(), d.GetEventMetadata().GetSequence(), version, d.GetEventMetadata().GetHubId()),
		ServiceMetadataUpdated: d.GetServiceMetadataUpdated(),
	}, true
}

type ServiceMetadataSnapshotTakenForCommand struct {
	userID string
	owner  string
	hubID  string
	*ServiceMetadataSnapshotTaken
}

func NewServiceMetadataSnapshotTakenForCommand(userID string, owner string, hubID string) *ServiceMetadataSnapshotTakenForCommand {
	return &ServiceMetadataSnapshotTakenForCommand{
		ServiceMetadataSnapshotTaken: NewServiceMetadataSnapshotTaken(),
		userID:                       userID,
		owner:                        owner,
		hubID:                        hubID,
	}
}

func NewServiceMetadataSnapshotTaken() *ServiceMetadataSnapshotTaken {
	return &ServiceMetadataSnapshotTaken{
		ServiceMetadataUpdated: &ServiceMetadataUpdated{},
		EventMetadata:          &EventMetadata{},
	}
}
