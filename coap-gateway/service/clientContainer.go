package service

import (
	"sync"
)

// ClientContainer client <-> server connections.
type ClientContainer struct {
	sessions map[string]*Client
	mutex    sync.Mutex
}

// Add sets client to container.
func (c *ClientContainer) Add(remoteAddr string, client *Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessions[remoteAddr] = client
}

// Find gets client from container.
func (c *ClientContainer) Find(remoteAddr string) (*Client, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if session, ok := c.sessions[remoteAddr]; ok {
		return session, true
	}
	return nil, false
}

// Pop finds and removes client from container.
func (c *ClientContainer) Pop(remoteAddr string) (*Client, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	defer delete(c.sessions, remoteAddr)
	if client, ok := c.sessions[remoteAddr]; ok {
		return client, true
	}
	return nil, false
}
