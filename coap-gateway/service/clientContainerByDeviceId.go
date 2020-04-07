package service

import (
	"sync"
)

//clientContainerByDeviceId client <-> server connections
type clientContainerByDeviceId struct {
	sessions map[string]*Client
	mutex    sync.Mutex
}

func NewClientContainerByDeviceId() *clientContainerByDeviceId {
	return &clientContainerByDeviceId{sessions: make(map[string]*Client)}
}

func (c *clientContainerByDeviceId) Add(deviceId string, session *Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessions[deviceId] = session
}

func (c *clientContainerByDeviceId) Find(deviceId string) *Client {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if session, ok := c.sessions[deviceId]; ok {
		return session
	}
	return nil
}

func (c *clientContainerByDeviceId) Remove(deviceId string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.sessions, deviceId)
}
