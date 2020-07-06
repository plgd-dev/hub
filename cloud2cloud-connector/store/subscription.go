package store

type Type string

const (
	Type_Devices  Type = "devices"
	Type_Device   Type = "device"
	Type_Resource Type = "resource"
)

type Subscription struct {
	ID              string
	Type            Type
	LinkedAccountID string
	LinkedCloudID   string
	DeviceID        string
	Href            string
	SigningSecret   string
	CorrelationID   string
}
