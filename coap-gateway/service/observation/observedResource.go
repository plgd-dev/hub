package observation

import (
	"context"
	"encoding/base64"
	"sort"
	"strings"
	"sync"

	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"go.uber.org/atomic"
)

type Observation = interface {
	Cancel(context.Context, ...message.Option) error
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

const (
	// maxURIQueryLen is the maximum length of a URI query. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxURIQueryLen = 255
	// maxETagLen is the maximum length of an ETag. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxETagLen = 8
	// prefixQueryIncChanged is the prefix of the URI query for the "incremental changed" option. See https://docs.plgd.dev/docs/features/control-plane/entity-tag/#etag-batch-interface-for-oicres
	prefixQueryIncChanges = "incChanges="
)

func encodeETagsForIncrementChanges(etags [][]byte) []string {
	if len(etags) < 1 {
		return nil
	}
	etagsStr := make([]string, 0, (len(etags)/15)+1)
	var b strings.Builder
	for _, etag := range etags {
		if len(etag) > maxETagLen {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(prefixQueryIncChanges)
		} else {
			b.WriteString(",")
		}
		b.WriteString(base64.RawURLEncoding.EncodeToString(etag))
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

func (r *observedResource) isBatchObservation() bool {
	return r.resInterface == interfaces.OC_IF_B
}

func (r *observedResource) toCoapOptions(etags [][]byte) []message.Option {
	opts := make([]message.Option, 0, 2)
	if len(etags) > 0 {
		opts = append(opts, message.Option{
			ID:    message.ETag,
			Value: etags[0],
		})
		etags = etags[1:]
	}
	if r.Interface() != "" {
		opts = append(opts, message.Option{
			ID:    message.URIQuery,
			Value: []byte("if=" + r.Interface()),
		})
	}

	if r.isBatchObservation() {
		for _, q := range encodeETagsForIncrementChanges(etags) {
			opts = append(opts, message.Option{
				ID:    message.URIQuery,
				Value: []byte(q),
			})
		}
	}
	return opts
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
