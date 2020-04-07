package service

import (
	"sync"
)

//ClientContainer client <-> server connections
type ClientContainer struct {
	sessions map[string]*Client
	mutex    sync.Mutex
}

func (c *ClientContainer) Add(remoteAddr string, client *Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessions[remoteAddr] = client
}

func (c *ClientContainer) Find(remoteAddr string) *Client {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if session, ok := c.sessions[remoteAddr]; ok {
		return session
	}
	return nil
}

func (c *ClientContainer) Pop(remoteAddr string) (*Client, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	defer delete(c.sessions, remoteAddr)
	if client, ok := c.sessions[remoteAddr]; ok {
		return client, true
	}
	return nil, false
}
