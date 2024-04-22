package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeServiceMetadataUpdated = "ServiceMetadataUpdated"

func (d *ServiceMetadataUpdated) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *ServiceMetadataUpdated) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *ServiceMetadataUpdated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *ServiceMetadataUpdated) EventType() string {
	return eventTypeServiceMetadataUpdated
}

func (d *ServiceMetadataUpdated) AggregateID() string {
	return commands.MakeServicesResourceUUID(d.GetEventMetadata().GetHubId()).String()
}

func (d *ServiceMetadataUpdated) GroupID() string {
	return d.GetEventMetadata().GetHubId()
}

func (d *ServiceMetadataUpdated) IsSnapshot() bool {
	return false
}

func (d *ServiceMetadataUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *ServiceMetadataUpdated) ETag() *eventstore.ETagData {
	return nil
}

func (d *ServiceMetadataUpdated) ServiceID() (string, bool) {
	return "", false
}

func (d *ServiceMetadataUpdated) Types() []string {
	return nil
}

func (d *ServiceMetadataUpdated) CopyData(event *ServiceMetadataUpdated) {
	d.EventMetadata = event.GetEventMetadata()
	d.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()

	sh := &ServicesHeartbeat{}
	sh.CopyData(event.GetServicesHeartbeat())
	d.ServicesHeartbeat = sh
}

func (d *ServiceMetadataUpdated) CheckInitialized() bool {
	return d.GetServicesHeartbeat() != nil &&
		d.GetEventMetadata() != nil
}

func (s *ServicesHeartbeat) CopyData(s1 *ServicesHeartbeat) {
	s.Valid = s1.GetValid()
	s.Expired = s1.GetExpired()
}

func equalServicesHeartbeates(v1, v2 []*ServicesHeartbeat_Heartbeat) bool {
	if len(v1) != len(v2) {
		return false
	}
	for idx := range v1 {
		if v1[idx].GetServiceId() != v2[idx].GetServiceId() {
			return false
		}
		if v1[idx].GetValidUntil() != v2[idx].GetValidUntil() {
			return false
		}
	}
	return true
}

func (s *ServicesHeartbeat) Equal(upd *ServicesHeartbeat) bool {
	if !equalServicesHeartbeates(s.GetValid(), upd.GetValid()) {
		return false
	}
	if !equalServicesHeartbeates(s.GetExpired(), upd.GetExpired()) {
		return false
	}
	return true
}

// Equal checks if two ServiceMetadataUpdated events are equal.
func (d *ServiceMetadataUpdated) Equal(upd *ServiceMetadataUpdated) bool {
	if d.GetServicesHeartbeat() == nil {
		return false
	}
	return d.GetServicesHeartbeat().Equal(upd.GetServicesHeartbeat())
}
