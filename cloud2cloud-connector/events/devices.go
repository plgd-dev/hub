package events

type Device struct {
	ID string `json:"di"`
}

type DevicesOnline []Device
type DevicesOffline []Device
type DevicesRegistered []Device
type DevicesUnregistered []Device
