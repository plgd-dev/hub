package events

import (
	"bytes"
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceChanged = "resourcechanged"

func (rc *ResourceChanged) Version() uint64 {
	return rc.GetEventMetadata().GetVersion()
}

func (rc *ResourceChanged) Marshal() ([]byte, error) {
	return proto.Marshal(rc)
}

func (rc *ResourceChanged) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, rc)
}

func (rc *ResourceChanged) EventType() string {
	return eventTypeResourceChanged
}

func (rc *ResourceChanged) AggregateID() string {
	return rc.GetResourceId().ToUUID().String()
}

func (rc *ResourceChanged) GroupID() string {
	return rc.GetResourceId().GetDeviceId()
}

func (rc *ResourceChanged) IsSnapshot() bool {
	return false
}

func (rc *ResourceChanged) ETag() *eventstore.ETagData {
	if len(rc.GetEtag()) == 0 {
		return nil
	}
	return &eventstore.ETagData{
		ETag:      rc.GetEtag(),
		Timestamp: rc.GetEventMetadata().GetTimestamp(),
	}
}

func (rc *ResourceChanged) Timestamp() time.Time {
	return pkgTime.Unix(0, rc.GetEventMetadata().GetTimestamp())
}

func (rc *ResourceChanged) ServiceID() (string, bool) {
	return "", false
}

func (rc *ResourceChanged) Types() []string {
	return rc.GetResourceTypes()
}

func (rc *ResourceChanged) CopyData(event *ResourceChanged) {
	rc.ResourceId = event.GetResourceId()
	rc.Content = event.GetContent()
	rc.AuditContext = event.GetAuditContext()
	rc.EventMetadata = event.GetEventMetadata()
	rc.Status = event.GetStatus()
	rc.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
	rc.Etag = event.GetEtag()
	rc.ResourceTypes = event.GetResourceTypes()
}

func (rc *ResourceChanged) Clone() *ResourceChanged {
	return proto.Clone(rc).(*ResourceChanged)
}

func (rc *ResourceChanged) CheckInitialized() bool {
	return rc.GetResourceId() != nil &&
		rc.GetContent() != nil &&
		rc.GetAuditContext() != nil &&
		rc.GetEventMetadata() != nil &&
		rc.GetStatus() != commands.Status(0)
}

func (rc *ResourceChanged) Equal(changed *ResourceChanged) bool {
	if rc.GetStatus() != changed.GetStatus() {
		return false
	}

	if rc.GetContent().GetCoapContentFormat() != changed.GetContent().GetCoapContentFormat() ||
		rc.GetContent().GetContentType() != changed.GetContent().GetContentType() ||
		!bytes.Equal(rc.GetContent().GetData(), changed.GetContent().GetData()) {
		return false
	}

	if rc.GetAuditContext().GetUserId() != changed.GetAuditContext().GetUserId() {
		return false
	}
	if !slices.Equal(rc.GetResourceTypes(), changed.GetResourceTypes()) {
		return false
	}

	return true
}
