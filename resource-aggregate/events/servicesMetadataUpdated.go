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

func (s *ServicesStatus) Equal(upd *ServicesStatus) bool {
	if len(s.GetOnline()) != len(upd.GetOnline()) {
		return false
	}
	for idx := range s.GetOnline() {
		if s.GetOnline()[idx].GetId() != upd.GetOnline()[idx].GetId() {
			return false
		}
		if s.GetOnline()[idx].GetOnlineValidUntil() != upd.GetOnline()[idx].GetOnlineValidUntil() {
			return false
		}
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
