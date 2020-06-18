package store

type Type string

const (
	Type_Devices      Type = "devices"
	Type_Device       Type = "device"
	Type_Resource     Type = "resource"
	Type_PullDevices  Type = "pull_devices"
	Type_PullDevice   Type = "pull_device"
	Type_PullResource Type = "pull_resource"
)

type Subscription struct {
	SubscriptionID  string
	Type            Type
	LinkedAccountID string
	DeviceID        string
	Href            string
	SigningSecret   string
}
