package observation

import (
	"sort"
	"sync"

	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/coap-gateway/resource"
	"github.com/plgd-dev/hub/pkg/log"
)

// Thread-safe wrapper with additional data for *tcp.Observation.
type observedResource struct {
	href         string
	resInterface string

	mutex       sync.Mutex
	observation *tcp.Observation
}

func NewObservedResource(href, resInterface string) *observedResource {
	return &observedResource{
		href:         href,
		resInterface: resInterface,
	}
}

func (r *observedResource) Equals(res *observedResource) bool {
	return r.href == res.href
}

func (r *observedResource) Href() string {
	return r.href
}

func (r *observedResource) InstanceID() int64 {
	return resource.GetInstanceID(r.Href())
}

func (r *observedResource) Interface() string {
	return r.resInterface
}

func (r *observedResource) SetObservation(o *tcp.Observation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.observation = o
}

func (r *observedResource) PopObservation() *tcp.Observation {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	o := r.observation
	r.observation = nil
	return o
}

type observedResources []*observedResource

func (o observedResources) containsResourceWithHref(href string) bool {
	i := o.searchByHref(href)
	return i < len(o) && o[i].Href() == href
}

func (o observedResources) searchByHref(href string) int {
	insID := resource.GetInstanceID(href)
	return o.searchByInstanceID(insID)
}

func (o observedResources) searchByInstanceID(instanceID int64) int {
	return sort.Search(len(o), func(i int) bool {
		return o[i].InstanceID() >= instanceID
	})
}

func (o observedResources) insert(rs ...*observedResource) observedResources {
	for _, v := range rs {
		i := o.searchByInstanceID(v.InstanceID())
		if i < len(o) && o[i].Equals(v) {
			continue
		}
		o = append(o, nil)
		copy(o[i+1:], o[i:])
		o[i] = v
	}
	return o
}

func (o observedResources) removeByHref(hrefs ...string) (new, removed observedResources) {
	removed = make(observedResources, 0, len(hrefs))
	for _, h := range hrefs {
		i := o.searchByHref(h)
		if i >= len(o) || o[i].Href() != h {
			log.Debugf("href(%v) not found", h)
			continue
		}
		removed = append(removed, o[i])
		copy(o[i:], o[i+1:])
	}

	if len(removed) > 0 {
		return o[:len(o)-len(removed)], removed
	}
	return o, nil
}
