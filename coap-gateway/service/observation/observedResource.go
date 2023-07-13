package observation

import (
	"context"
	"encoding/hex"
	"sort"
	"strings"
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
	etags        [][]byte
	synced       atomic.Bool
	isObservable bool
	private      struct { // guarded by mutex
		mutex       sync.Mutex
		observation Observation
	}
}

func newObservedResource(href, resInterface string, etags [][]byte, isObservable bool) *observedResource {
	return &observedResource{
		href:         href,
		resInterface: resInterface,
		isObservable: isObservable,
		etags:        etags,
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

func (r *observedResource) ETag() []byte {
	if len(r.etags) == 0 {
		return nil
	}
	return r.etags[0]
}

const (
	// maxURIQueryLen is the maximum length of a URI query. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxURIQueryLen = 255
	// maxETagLen is the maximum length of an ETag. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxETagLen = 8
	// prefixQueryIncChanged is the prefix of the URI query for the "incremental changed" option. See https://docs.plgd.dev/docs/features/control-plane/entity-tag/#etag-batch-interface-for-oicres
	prefixQueryIncChanged = "incChanged="
)

func (r *observedResource) EncodeETagsForIncrementChanged() []string {
	if len(r.etags) <= 1 {
		return nil
	}
	etags := r.etags[1:]
	etagsStr := make([]string, 0, (len(etags)/15)+1)
	var b strings.Builder
	for _, etag := range etags {
		if len(etag) > maxETagLen {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(prefixQueryIncChanged)
		} else {
			b.WriteString(",")
		}
		b.WriteString(hex.EncodeToString(etag))
		//
		if b.Len() >= maxURIQueryLen-(maxETagLen*2) {
			etagsStr = append(etagsStr, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		etagsStr = append(etagsStr, b.String())
	}
	return etagsStr
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
