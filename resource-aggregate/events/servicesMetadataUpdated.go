package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeServicesMetadataUpdated = "servicesmetadataupdated"

func (d *ServicesMetadataUpdated) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *ServicesMetadataUpdated) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *ServicesMetadataUpdated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *ServicesMetadataUpdated) EventType() string {
	return eventTypeServicesMetadataUpdated
}

func (d *ServicesMetadataUpdated) AggregateID() string {
	return commands.MakeServicesResourceUUID(d.GetEventMetadata().GetHubId()).String()
}

func (d *ServicesMetadataUpdated) GroupID() string {
	return d.GetEventMetadata().GetHubId()
}

func (d *ServicesMetadataUpdated) IsSnapshot() bool {
	return false
}

func (d *ServicesMetadataUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *ServicesMetadataUpdated) ETag() *eventstore.ETagData {
	return nil
}

func (d *ServicesMetadataUpdated) ServiceID() (string, bool) {
	return "", false
}

func (d *ServicesMetadataUpdated) CopyData(event *ServicesMetadataUpdated) {
	d.EventMetadata = event.GetEventMetadata()
	d.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()

	d.Status = &ServicesStatus{}
	d.Status.CopyData(event.GetStatus())
}

func (d *ServicesMetadataUpdated) CheckInitialized() bool {
	return d.GetStatus() != nil &&
		d.GetEventMetadata() != nil
}

func (s *ServicesStatus) CopyData(s1 *ServicesStatus) {
	s.Online = s1.GetOnline()
	s.Offline = s1.GetOffline()
}

func equalServicesStatuses(v1, v2 []*ServicesStatus_Status) bool {
	if len(v1) != len(v2) {
		return false
	}
	for idx := range v1 {
		if v1[idx].GetId() != v2[idx].GetId() {
			return false
		}
		if v1[idx].GetOnlineValidUntil() != v2[idx].GetOnlineValidUntil() {
			return false
		}
	}
	return true
}

func (s *ServicesStatus) Equal(upd *ServicesStatus) bool {
	if !equalServicesStatuses(s.GetOnline(), upd.GetOnline()) {
		return false
	}
	if !equalServicesStatuses(s.GetOffline(), upd.GetOffline()) {
		return false
	}
	return true
}

// Equal checks if two ServicesMetadataUpdated events are equal.
func (d *ServicesMetadataUpdated) Equal(upd *ServicesMetadataUpdated) bool {
	if d.GetStatus() == nil {
		return false
	}
	return d.GetStatus().Equal(upd.GetStatus())
}
