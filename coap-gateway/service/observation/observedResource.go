package observation

import (
	"context"
	"sync"

	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/pkg/sync/task/future"
)

type observedResource struct {
	href         string
	resInterface string
	opened       *future.Future
	setOpened    future.SetFunc

	mutex       sync.Mutex
	observation *tcp.Observation
}

func NewObservedResource(href, resInterface string) *observedResource {
	opened, setOpened := future.New()
	return &observedResource{
		href:         href,
		resInterface: resInterface,
		opened:       opened,
		setOpened:    setOpened,
	}
}

func (r *observedResource) Href() string {
	return r.href
}

func (r *observedResource) Interface() string {
	return r.resInterface
}

func (r *observedResource) IsOpened(ctx context.Context) (bool, error) {
	v, err := r.opened.Get(ctx)
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

func (r *observedResource) SetOpened(valid bool) {
	r.setOpened(valid, nil)
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
