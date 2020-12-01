package notification

import (
	"sync"

	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
)

type DeleteNotificationContainer struct {
	data sync.Map
}

func NewDeleteNotificationContainer() *DeleteNotificationContainer {
	return &DeleteNotificationContainer{}
}

func (c *DeleteNotificationContainer) Add(correlationID string) <-chan raEvents.ResourceDeleted {
	notify := make(chan raEvents.ResourceDeleted, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *DeleteNotificationContainer) Find(correlationID string) chan<- raEvents.ResourceDeleted {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan raEvents.ResourceDeleted)
}

func (c *DeleteNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
