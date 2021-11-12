package observation

import (
	"sync"

	"github.com/plgd-dev/go-coap/v2/tcp"
)

type ObservedResource struct {
	href       string
	observable bool

	mutex       sync.Mutex
	observation *tcp.Observation
}

func NewObservedResource(href string, observable bool) *ObservedResource {
	return &ObservedResource{
		href:       href,
		observable: observable,
	}
}

func (r *ObservedResource) Href() string {
	return r.href
}

func (r *ObservedResource) Observable() bool {
	return r.observable
}

func (r *ObservedResource) SetObservation(o *tcp.Observation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.observation = o
}

func (r *ObservedResource) PopObservation() *tcp.Observation {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	o := r.observation
	r.observation = nil
	return o
}
