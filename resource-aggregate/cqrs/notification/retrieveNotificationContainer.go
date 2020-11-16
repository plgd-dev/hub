package notification

import (
	"sync"

	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
)

type RetrieveNotificationContainer struct {
	data sync.Map
}

func NewRetrieveNotificationContainer() *RetrieveNotificationContainer {
	return &RetrieveNotificationContainer{}
}

func (c *RetrieveNotificationContainer) Add(correlationID string) <-chan raEvents.ResourceRetrieved {
	notify := make(chan raEvents.ResourceRetrieved, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *RetrieveNotificationContainer) Find(correlationID string) chan<- raEvents.ResourceRetrieved {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan raEvents.ResourceRetrieved)
}

func (c *RetrieveNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
