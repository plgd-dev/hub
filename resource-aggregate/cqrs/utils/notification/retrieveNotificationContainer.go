package notification

import (
	"sync"

	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type RetrieveNotificationContainer struct {
	data sync.Map
}

func NewRetrieveNotificationContainer() *RetrieveNotificationContainer {
	return &RetrieveNotificationContainer{}
}

func (c *RetrieveNotificationContainer) Add(correlationID string) <-chan events.ResourceRetrieved {
	notify := make(chan events.ResourceRetrieved, 1)
	c.data.Store(correlationID, notify)
	return notify
}

func (c *RetrieveNotificationContainer) Find(correlationID string) chan<- events.ResourceRetrieved {
	v, ok := c.data.Load(correlationID)
	if !ok {
		return nil
	}
	return v.(chan events.ResourceRetrieved)
}

func (c *RetrieveNotificationContainer) Remove(correlationID string) {
	c.data.Delete(correlationID)
}
