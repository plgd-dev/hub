package notification

import (
	"sync"

	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
)

type UpdateNotificationContainer struct {
	notifications map[string]chan raEvents.ResourceUpdated
	mutex         sync.Mutex
}

func NewUpdateNotificationContainer() *UpdateNotificationContainer {
	return &UpdateNotificationContainer{notifications: make(map[string]chan raEvents.ResourceUpdated)}
}

func (c *UpdateNotificationContainer) Add(correlationID string) <-chan raEvents.ResourceUpdated {
	notify := make(chan raEvents.ResourceUpdated, 1)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.notifications[correlationID] = notify
	return notify
}

func (c *UpdateNotificationContainer) Find(correlationID string) chan<- raEvents.ResourceUpdated {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if n, ok := c.notifications[correlationID]; ok {
		return n
	}
	return nil
}

func (c *UpdateNotificationContainer) Remove(correlationID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.notifications, correlationID)
}
