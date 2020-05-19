package service

import (
	"encoding/base64"
	"fmt"
	"sync"
)

type observeResource struct {
	remoteAddr string
	deviceID   string
	resourceID string
	token      []byte
	client     *Client

	mutex   sync.Mutex
	observe uint32
}

type observeResourceContainer struct {
	observersByResource   map[string]map[string]map[string]*observeResource //resourceID, remoteAddr, token
	observersByRemoteAddr map[string]map[string]*observeResource            //remoteAddr, token
	mutex                 sync.Mutex
}

func (r *observeResource) Observe() uint32 {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.observe++
	if r.observe >= 1<<24 || r.observe < 2 {
		r.observe = 2
	}
	return r.observe
}

func NewObserveResourceContainer() *observeResourceContainer {
	return &observeResourceContainer{
		observersByResource:   make(map[string]map[string]map[string]*observeResource),
		observersByRemoteAddr: make(map[string]map[string]*observeResource),
	}
}

func tokenToString(token []byte) string {
	return base64.StdEncoding.EncodeToString(token)
}

func (c *observeResourceContainer) prepareObserversByResourceLocked(resourceID, remoteAddr, token string) (map[string]*observeResource, error) {
	var ok bool
	var clients map[string]map[string]*observeResource
	var tokens map[string]*observeResource

	if clients, ok = c.observersByResource[resourceID]; !ok {
		clients = make(map[string]map[string]*observeResource)
		c.observersByResource[resourceID] = clients
	}
	if tokens, ok = clients[remoteAddr]; !ok {
		tokens = make(map[string]*observeResource)
		clients[remoteAddr] = tokens
	}
	if _, ok = tokens[token]; ok {
		return nil, fmt.Errorf("token already exists")
	}
	return tokens, nil
}

func (c *observeResourceContainer) prepareObserversByDeviceLocked(remoteAddr, token string) (map[string]*observeResource, error) {
	var ok bool
	var tokens map[string]*observeResource

	if tokens, ok = c.observersByRemoteAddr[remoteAddr]; !ok {
		tokens = make(map[string]*observeResource)
		c.observersByRemoteAddr[remoteAddr] = tokens
	}
	if _, ok = tokens[token]; ok {
		return nil, fmt.Errorf("token already exists")
	}
	return tokens, nil
}

func (c *observeResourceContainer) Add(observeResource observeResource) error {
	tokenStr := tokenToString(observeResource.token)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	byResource, err := c.prepareObserversByResourceLocked(observeResource.resourceID, observeResource.remoteAddr, tokenStr)
	if err != nil {
		return fmt.Errorf("cannot observe resource observersByResource[%v][%v][%v]: %v", observeResource.resourceID, observeResource.remoteAddr, observeResource.token, err)
	}
	byDevice, err := c.prepareObserversByDeviceLocked(observeResource.remoteAddr, tokenStr)
	if err != nil {
		c.removeByResourceLocked(observeResource.resourceID, observeResource.remoteAddr, tokenStr)
		//this cannot occurs - it mean that byResource and byDevice are unsync
		return fmt.Errorf("cannot observe resource observersByRemoteAddr[%v][%v]: %v", observeResource.remoteAddr, observeResource.token, err)
	}

	byResource[tokenStr] = &observeResource
	byDevice[tokenStr] = &observeResource
	return nil
}

func (c *observeResourceContainer) Find(resourceID string) []*observeResource {
	found := make([]*observeResource, 0, 128)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, clients := range c.observersByResource[resourceID] {
		for _, ob := range clients {
			found = append(found, ob)
		}
	}
	return found
}

func (c *observeResourceContainer) removeByResourceLocked(resourceID, remoteAddr, token string) error {
	found := false
	if clients, ok := c.observersByResource[resourceID]; ok {
		if tokens, ok := clients[remoteAddr]; ok {
			if _, ok := tokens[token]; ok {
				delete(tokens, token)
				found = true
			}
			if len(clients[remoteAddr]) == 0 {
				delete(clients, remoteAddr)
			}
		}
		if len(c.observersByResource[resourceID]) == 0 {
			delete(c.observersByResource, resourceID)
		}
	}
	if !found {
		return fmt.Errorf("not found")
	}
	return nil
}

func (c *observeResourceContainer) RemoveByResource(resourceID, remoteAddr string, token []byte) error {
	tokenStr := tokenToString(token)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.removeByResourceLocked(resourceID, remoteAddr, tokenStr)
	if err != nil {
		return fmt.Errorf("cannot remove observer of resource: %v", err)
	}
	if tokens, ok := c.observersByRemoteAddr[remoteAddr]; ok {
		if _, ok := tokens[tokenStr]; ok {
			delete(tokens, tokenStr)
			if len(tokens) == 0 {
				delete(c.observersByRemoteAddr, remoteAddr)
			}
			return nil
		}
		return fmt.Errorf("unstable container - observersByRemoteAddr[%v][%v]", remoteAddr, token)
	} else {
		return fmt.Errorf("unstable container - observersByRemoteAddr[%v]", remoteAddr)
	}
}

func (c *observeResourceContainer) PopByRemoteAddr(remoteAddr string) ([]*observeResource, error) {
	poped := make([]*observeResource, 0, 32)
	var tokens map[string]*observeResource
	var ok bool

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if tokens, ok = c.observersByRemoteAddr[remoteAddr]; !ok {
		return nil, fmt.Errorf("not found")
	}
	delete(c.observersByRemoteAddr, remoteAddr)
	var errors []error
	for token, obs := range tokens {
		err := c.removeByResourceLocked(obs.resourceID, remoteAddr, token)
		if err != nil {
			errors = append(errors, fmt.Errorf("observersByResource[%v][%v][%v]:%v", obs.resourceID, remoteAddr, token, err))
		}
		poped = append(poped, obs)
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("unstable container: %v", errors)
	}
	return poped, nil
}

func (c *observeResourceContainer) PopByRemoteAddrToken(remoteAddr string, token []byte) (*observeResource, error) {
	var obs *observeResource
	var tokens map[string]*observeResource
	var ok bool

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if tokens, ok = c.observersByRemoteAddr[remoteAddr]; !ok {
		return obs, fmt.Errorf("remote address not found")
	}

	tokenStr := tokenToString(token)
	if obs, ok = tokens[tokenStr]; !ok {
		return obs, fmt.Errorf("token not found")
	}
	delete(tokens, tokenStr)

	err := c.removeByResourceLocked(obs.resourceID, remoteAddr, tokenStr)
	if err != nil {
		return obs, fmt.Errorf("unstable container: observersByResource[%v][%v][%v]:%v", obs.resourceID, remoteAddr, token, err)
	}
	return obs, nil
}
