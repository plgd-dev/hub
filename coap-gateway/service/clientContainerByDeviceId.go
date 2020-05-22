package service

import (
	"sync"
)

// clientContainerByDeviceID client <-> server connections
type clientContainerByDeviceID struct {
	sessions map[string]*Client
	mutex    sync.Mutex
}

func newClientContainerByDeviceID() *clientContainerByDeviceID {
	return &clientContainerByDeviceID{sessions: make(map[string]*Client)}
}

func (c *clientContainerByDeviceID) Add(deviceID string, session *Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessions[deviceID] = session
}

func (c *clientContainerByDeviceID) Find(deviceID string) *Client {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if session, ok := c.sessions[deviceID]; ok {
		return session
	}
	return nil
}

func (c *clientContainerByDeviceID) Remove(deviceID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.sessions, deviceID)
}
