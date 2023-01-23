package observation

import (
	"context"
	"sort"
	"sync"

	"go.uber.org/atomic"
)

type Observation = interface {
	Cancel(context.Context) error
	Canceled() bool
}

// Thread-safe wrapper with additional data for *tcp.Observation.
type observedResource struct {
	href         string
	resInterface string
	synced       atomic.Bool
	isObservable bool
	private      struct { // guarded by mutex
		mutex       sync.Mutex
		observation Observation
	}
}

func newObservedResource(href, resInterface string, isObservable bool) *observedResource {
	return &observedResource{
		href:         href,
		resInterface: resInterface,
		isObservable: isObservable,
	}
}

func (r *observedResource) Equals(href string) bool {
	return r.href == href
}

func (r *observedResource) Href() string {
	return r.href
}

func (r *observedResource) Interface() string {
	return r.resInterface
}

func (r *observedResource) SetObservation(o Observation) {
	r.private.mutex.Lock()
	defer r.private.mutex.Unlock()
	r.private.observation = o
}

func (r *observedResource) PopObservation() Observation {
	r.private.mutex.Lock()
	defer r.private.mutex.Unlock()
	o := r.private.observation
	r.private.observation = nil
	return o
}

type observedResources []*observedResource

func (o observedResources) contains(href string) bool {
	i := o.search(href)
	return i < len(o) && o[i].Href() == href
}

func (o observedResources) search(href string) int {
	return sort.Search(len(o), func(i int) bool {
		return o[i].Href() >= href
	})
}

func (o observedResources) insert(rs ...*observedResource) observedResources {
	for _, v := range rs {
		i := o.search(v.Href())
		if i < len(o) && o[i].Equals(v.Href()) {
			continue
		}
		o = append(o, nil)
		copy(o[i+1:], o[i:])
		o[i] = v
	}
	return o
}

func (o observedResources) removeByHref(hrefs ...string) (retained, removed observedResources) {
	removed = make(observedResources, 0, len(hrefs))
	for _, h := range hrefs {
		i := o.search(h)
		if i >= len(o) || o[i].Href() != h {
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
