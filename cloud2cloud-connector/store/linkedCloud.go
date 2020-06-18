package store

import (
	"github.com/go-ocf/cloud/authorization/oauth"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
)

type Events struct {
	Devices  []events.EventType `json:"Devices"`
	Device   []events.EventType `json:"Device"`
	Resource []events.EventType `json:"Resource"`
}

type LinkedCloud struct {
	ID                           string       `json:"Id" bson:"_id"`
	Name                         string       `json:"Name"`
	OAuth                        oauth.Config `json:"OAuth"`
	RootCA                       []string     `json:"RootCA"`
	SupportedSubscriptionsEvents Events       `json:"SupportedSubscriptionEvents"`
}
