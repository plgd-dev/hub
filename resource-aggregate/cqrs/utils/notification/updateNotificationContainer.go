package notification

import (
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type UpdateNotificationContainer struct {
	data sync.Map
}

func NewUpdateNotificationContainer() *UpdateNotificationContainer {
	return &UpdateNotificationContainer{}
}

func (c *UpdateNotificationContainer) Add(correlationID string) <-chan *events.ResourceUpdated {
	notify := make(chan *events.ResourceUpdated, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *UpdateNotificationContainer) Find(correlationID string) chan<- *events.ResourceUpdated {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan *events.ResourceUpdated)
}

func (c *UpdateNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
