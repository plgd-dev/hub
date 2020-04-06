package notification

import (
	"sync"

	raEvents "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/events"
)

type RetrieveNotificationContainer struct {
	notifications map[string]chan raEvents.ResourceRetrieved
	mutex         sync.Mutex
}

func NewRetrieveNotificationContainer() *RetrieveNotificationContainer {
	return &RetrieveNotificationContainer{notifications: make(map[string]chan raEvents.ResourceRetrieved)}
}

func (c *RetrieveNotificationContainer) Add(correlationID string) <-chan raEvents.ResourceRetrieved {
	notify := make(chan raEvents.ResourceRetrieved, 1)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.notifications[correlationID] = notify
	return notify
}

func (c *RetrieveNotificationContainer) Find(correlationID string) chan<- raEvents.ResourceRetrieved {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if n, ok := c.notifications[correlationID]; ok {
		return n
	}
	return nil
}

func (c *RetrieveNotificationContainer) Remove(correlationID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.notifications, correlationID)
}
