package notification

import (
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type DeleteNotificationContainer struct {
	data sync.Map
}

func NewDeleteNotificationContainer() *DeleteNotificationContainer {
	return &DeleteNotificationContainer{}
}

func (c *DeleteNotificationContainer) Add(correlationID string) <-chan events.ResourceDeleted {
	notify := make(chan events.ResourceDeleted, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *DeleteNotificationContainer) Find(correlationID string) chan<- events.ResourceDeleted {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan events.ResourceDeleted)
}

func (c *DeleteNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
