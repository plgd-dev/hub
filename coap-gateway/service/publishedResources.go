package service

import (
	"sync"

	"github.com/plgd-dev/hub/coap-gateway/resource"
	"github.com/plgd-dev/hub/pkg/strings"
)

// Thread-safe container of published resource hrefs
type publishedResources struct {
	lock      sync.Mutex
	resources strings.SortedSlice
}

func (p *publishedResources) Add(href ...string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.resources = p.resources.Insert(href...)
}

func (p *publishedResources) Remove(href ...string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.resources = p.resources.Remove(href...)
}

// Get list of published resources
//
// Empty instanceIDs parameter is ignored and function will return all resources. Otherwise only
// resources with instanceID value contained in the instanceIDs array are returned.
func (p *publishedResources) Get(instanceIDs []int64) []string {
	getAllDeviceIDMatches := len(instanceIDs) == 0
	if getAllDeviceIDMatches {
		p.lock.Lock()
		defer p.lock.Unlock()
		return append([]string(nil), p.resources...)
	}

	uniqueInstanceIDs := make(map[int64]struct{})
	for _, v := range instanceIDs {
		uniqueInstanceIDs[v] = struct{}{}
	}

	p.lock.Lock()
	resouceHrefs := append([]string(nil), p.resources...)
	p.lock.Unlock()

	hrefs := make([]string, 0, len(instanceIDs))
	for _, href := range resouceHrefs {
		instanceID := resource.GetInstanceID(href)
		if _, ok := uniqueInstanceIDs[instanceID]; ok {
			hrefs = append(hrefs, href)
			delete(uniqueInstanceIDs, instanceID)
		}
		if len(uniqueInstanceIDs) == 0 {
			break
		}
	}

	return hrefs
}
