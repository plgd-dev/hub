package utils

import (
	"fmt"
	"time"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

const (
	DeviceIDKey = "deviceId"
	HrefIDKey   = "hrefId"
)

const (
	Devices                                            = "devices"
	PlgdOwnersOwnerDevices                             = isEvents.PlgdOwnersOwner + "." + Devices
	PlgdOwnersOwnerDevicesDevice                       = PlgdOwnersOwnerDevices + ".{" + DeviceIDKey + "}"
	PlgdOwnersOwnerDevicesDeviceResourceLinks          = PlgdOwnersOwnerDevicesDevice + ".resource-links"
	PlgdOwnersOwnerDevicesDeviceResourceLinksEvent     = PlgdOwnersOwnerDevicesDeviceResourceLinks + ".{" + isEvents.EventTypeKey + "}"
	PlgdOwnersOwnerDevicesDeviceMetadata               = PlgdOwnersOwnerDevicesDevice + ".metadata"
	PlgdOwnersOwnerDevicesDeviceMetadataEvent          = PlgdOwnersOwnerDevicesDeviceMetadata + ".{" + isEvents.EventTypeKey + "}"
	PlgdOwnersOwnerDevicesDeviceResources              = PlgdOwnersOwnerDevicesDevice + ".resources"
	PlgdOwnersOwnerDevicesDeviceResourcesResource      = PlgdOwnersOwnerDevicesDeviceResources + ".{" + HrefIDKey + "}"
	PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent = PlgdOwnersOwnerDevicesDeviceResourcesResource + ".{" + isEvents.EventTypeKey + "}"
)

func HrefToID(href string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(href))
}

func WithHrefId(hrefId string) func(values map[string]string) {
	return func(values map[string]string) {
		values[HrefIDKey] = hrefId
	}
}

func WithDeviceID(deviceID string) func(values map[string]string) {
	return func(values map[string]string) {
		values[DeviceIDKey] = deviceID
	}
}

func GetDeviceSubject(owner, deviceID string) []string {
	return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDevice, isEvents.WithOwner(owner), WithDeviceID(deviceID)) + ".>"}
}

func GetDeviceMetadataEventSubject(owner, deviceID, eventType string) []string {
	return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner(owner), WithDeviceID(deviceID), isEvents.WithEventType(eventType))}
}

func GetResourceEventSubject(owner string, resourceID *commands.ResourceId, eventType string) []string {
	return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithOwner(owner), WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType(eventType), WithHrefId(HrefToID(resourceID.GetHref()).String()))}
}

func GetPublishSubject(owner string, event eventbus.Event) []string {
	switch event.EventType() {
	case (&events.ResourceLinksPublished{}).EventType(), (&events.ResourceLinksUnpublished{}).EventType(), (&events.ResourceLinksSnapshotTaken{}).EventType():
		return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithOwner(owner), WithDeviceID(event.GroupID()), isEvents.WithEventType(event.EventType()))}
	case (&events.DeviceMetadataUpdatePending{}).EventType(), (&events.DeviceMetadataUpdated{}).EventType(), (&events.DeviceMetadataSnapshotTaken{}).EventType():
		return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner(owner), WithDeviceID(event.GroupID()), isEvents.WithEventType(event.EventType()))}
	}
	if ev, ok := event.(interface{ GetResourceId() *commands.ResourceId }); ok {
		return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithOwner(owner), WithDeviceID(event.GroupID()), WithHrefId(HrefToID(ev.GetResourceId().GetHref()).String()), isEvents.WithEventType(event.EventType()))}
	}
	return nil
}

func TimeNowMs() uint64 {
	now := time.Now()
	unix := now.UnixNano()
	return uint64(unix / int64(time.Millisecond))
}

type ProtobufMarshaler interface {
	Marshal() ([]byte, error)
}

type ProtobufUnmarshaler interface {
	Unmarshal([]byte) error
}

func Marshal(v interface{}) ([]byte, error) {
	if p, ok := v.(ProtobufMarshaler); ok {
		src, err := p.Marshal()
		if err != nil {
			return nil, fmt.Errorf("cannot marshal event: %w", err)
		}
		dst := make([]byte, 1024)
		return snappy.Encode(dst, src), nil
	}
	return nil, fmt.Errorf("marshal is not supported by %T", v)
}

func Unmarshal(b []byte, v interface{}) error {
	if p, ok := v.(ProtobufUnmarshaler); ok {
		dst := make([]byte, 1024)
		dst, err := snappy.Decode(dst, b)
		if err != nil {
			return fmt.Errorf("cannot decode buffer: %w", err)
		}
		return p.Unmarshal(dst)
	}
	return fmt.Errorf("unmarshal is not supported by %T", v)
}
