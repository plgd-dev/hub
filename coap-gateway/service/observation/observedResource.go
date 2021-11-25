package observation

import (
	"sync"

	"github.com/plgd-dev/go-coap/v2/tcp"
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

func (r *observedResource) Href() string {
	return r.href
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
