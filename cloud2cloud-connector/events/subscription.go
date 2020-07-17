package events

type EventType string

const (
	// resource
	EventType_ResourceChanged EventType = "resource_contentchanged"

	// device
	EventType_ResourcesPublished   EventType = "resources_published"
	EventType_ResourcesUnpublished EventType = "resources_unpublished"

	// devices
	EventType_DevicesOnline       EventType = "devices_online"
	EventType_DevicesOffline      EventType = "devices_offline"
	EventType_DevicesRegistered   EventType = "devices_registered"
	EventType_DevicesUnregistered EventType = "devices_unregistered"

	// among all
	EventType_SubscriptionCanceled EventType = "subscription_cancelled"
)

var AllDevicesEvents = []EventType{EventType_DevicesOnline, EventType_DevicesOffline, EventType_DevicesRegistered, EventType_DevicesUnregistered}
var AllDeviceEvents = []EventType{EventType_ResourcesPublished, EventType_ResourcesUnpublished}
var AllResourceEvents = []EventType{EventType_ResourceChanged}

type EventTypes []EventType

func (e EventTypes) Has(ev EventType) bool {
	for _, v := range e {
		if v == ev {
			return true
		}
	}
	return false
}

type SubscriptionRequest struct {
	URL           string      `json:"eventsUrl"`
	EventTypes    []EventType `json:"eventTypes"`
	SigningSecret string      `json:"signingSecret"`
}

type SubscriptionResponse struct {
	SubscriptionId string `json:"subscriptionId"`
}
