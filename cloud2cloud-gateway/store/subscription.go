package store

import (
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
)

type Type string

const (
	Type_Devices  Type = "devices"
	Type_Device   Type = "device"
	Type_Resource Type = "resource"
)

type Subscription struct {
	ID             string // Id
	URL            string // href
	CorrelationID  string // uuid
	Type           Type
	ContentType    string // application/json or application/vnd.ocf+cbor
	EventTypes     []events.EventType
	DeviceID       string // filled for device and resource events
	Href           string // filled for resource events
	SequenceNumber uint64
	UserID         string
	SigningSecret  string
}

type DevicesSubscription struct {
	// EventTypes = [devices_registered, devices_unregistered, devices_online, devices_offline]
	Subscription
	AccessToken           string
	LastDevicesRegistered events.DevicesRegistered
	LastDevicesOnline     events.DevicesOnline
	LastDevicesOffline    events.DevicesOffline
	LastCheck             time.Time
}
