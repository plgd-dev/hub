package store

import (
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
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
	Accept         []string // application/json or application/vnd.ocf+cbor
	EventTypes     events.EventTypes
	DeviceID       string // filled for device and resource events
	Href           string // filled for resource events
	SequenceNumber uint64
	UserID         string
	SigningSecret  string
}
