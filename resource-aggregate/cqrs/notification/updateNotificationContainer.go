package notification

import (
	"sync"

	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
)

type UpdateNotificationContainer struct {
	data sync.Map
}

func NewUpdateNotificationContainer() *UpdateNotificationContainer {
	return &UpdateNotificationContainer{}
}

func (c *UpdateNotificationContainer) Add(correlationID string) <-chan raEvents.ResourceUpdated {
	notify := make(chan raEvents.ResourceUpdated, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *UpdateNotificationContainer) Find(correlationID string) chan<- raEvents.ResourceUpdated {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan raEvents.ResourceUpdated)
}

func (c *UpdateNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
