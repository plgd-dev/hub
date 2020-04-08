package events

type EventType string

const (
	EventType_ResourceChanged      EventType = "resource_contentchanged"
	EventType_ResourcesPublished   EventType = "resources_published"
	EventType_ResourcesUnpublished EventType = "resources_unpublished"
	EventType_DevicesOnline        EventType = "devices_online"
	EventType_DevicesOffline       EventType = "devices_offline"
	EventType_DevicesRegistered    EventType = "devices_registered"
	EventType_DevicesUnregistered  EventType = "devices_unregistered"
	EventType_SubscriptionCanceled EventType = "subscription_canceled"
)

type SubscriptionRequest struct {
	URL           string      `json:"eventsUrl"`
	EventTypes    []EventType `json:"eventTypes"`
	SigningSecret string      `json:"signingSecret"`
}

type SubscriptionResponse struct {
	SubscriptionId string `json:"subscriptionId"`
}
