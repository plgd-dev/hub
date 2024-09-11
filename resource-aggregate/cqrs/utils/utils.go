package utils

import (
	"fmt"
	"time"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const (
	DeviceIDKey         = "deviceId"
	HrefIDKey           = "hrefId"
	LeadResourceTypeKey = "leadResourceType"
	LeadResourcePrefix  = "leadrt"
)

const (
	Devices                                                            = "devices"
	PlgdOwnersOwnerDevices                                             = isEvents.PlgdOwnersOwner + "." + Devices
	PlgdOwnersOwnerDevicesDevice                                       = PlgdOwnersOwnerDevices + ".{" + DeviceIDKey + "}"
	PlgdOwnersOwnerDevicesDeviceResourceLinks                          = PlgdOwnersOwnerDevicesDevice + ".resource-links"
	PlgdOwnersOwnerDevicesDeviceResourceLinksEvent                     = PlgdOwnersOwnerDevicesDeviceResourceLinks + ".{" + isEvents.EventTypeKey + "}"
	PlgdOwnersOwnerDevicesDeviceMetadata                               = PlgdOwnersOwnerDevicesDevice + ".metadata"
	PlgdOwnersOwnerDevicesDeviceMetadataEvent                          = PlgdOwnersOwnerDevicesDeviceMetadata + ".{" + isEvents.EventTypeKey + "}"
	PlgdOwnersOwnerDevicesDeviceResources                              = PlgdOwnersOwnerDevicesDevice + ".resources"
	PlgdOwnersOwnerDevicesDeviceResourcesResource                      = PlgdOwnersOwnerDevicesDeviceResources + ".{" + HrefIDKey + "}"
	PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent                 = PlgdOwnersOwnerDevicesDeviceResourcesResource + ".{" + isEvents.EventTypeKey + "}"
	PlgdOwnersOwnerDevicesDeviceResourcesResourceEventLeadResourceType = PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent + "." + LeadResourcePrefix + ".{" + LeadResourceTypeKey + "}"
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

func WithLeadResourceType(leadResourceType string) func(values map[string]string) {
	return func(values map[string]string) {
		values[LeadResourceTypeKey] = leadResourceType
	}
}

func GetDeviceSubject(owner, deviceID string) []string {
	return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDevice, isEvents.WithOwner(owner), WithDeviceID(deviceID)) + ".>"}
}

func GetDeviceMetadataEventSubject(owner, deviceID, eventType string) []string {
	return []string{isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner(owner), WithDeviceID(deviceID), isEvents.WithEventType(eventType))}
}

func GetResourceEventSubjects(owner string, resourceID *commands.ResourceId, eventType string, leadResourceTypeEnabled bool) []string {
	subject := isEvents.ToSubject(PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithOwner(owner), WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType(eventType), WithHrefId(GetSubjectHrefID(resourceID.GetHref())))
	if !leadResourceTypeEnabled {
		return []string{subject}
	}
	return []string{subject + ".>"}
}

func GetSubjectHrefID(href string) string {
	switch href {
	case "", "*":
		return "*"
	default:
		return HrefToID(href).String()
	}
}

func TimeNowMs() uint64 {
	unix := time.Now().UnixNano()
	return math.CastTo[uint64](unix / int64(time.Millisecond))
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
