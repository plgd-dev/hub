package notification

import (
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type CreateNotificationContainer struct {
	data sync.Map
}

func NewCreateNotificationContainer() *CreateNotificationContainer {
	return &CreateNotificationContainer{}
}

func (c *CreateNotificationContainer) Add(correlationID string) <-chan *events.ResourceCreated {
	notify := make(chan *events.ResourceCreated, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *CreateNotificationContainer) Find(correlationID string) chan<- *events.ResourceCreated {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan *events.ResourceCreated)
}

func (c *CreateNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
