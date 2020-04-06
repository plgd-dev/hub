package projection

import (
	"fmt"
	"sync"
)

type RefCountMap struct {
	countsLock sync.Mutex
	counts     map[string]int
}

func NewRefCountMap() *RefCountMap {
	return &RefCountMap{counts: make(map[string]int)}
}

func (p *RefCountMap) Inc(id string, create bool) (created bool, err error) {
	var ok bool
	var count int

	p.countsLock.Lock()
	defer p.countsLock.Unlock()
	if count, ok = p.counts[id]; !ok && !create {
		return false, fmt.Errorf("cannot increment reference counter: not found")
	}
	count++
	p.counts[id] = count

	if count == 1 {
		created = true
	}

	return created, nil
}

func (p *RefCountMap) Dec(id string) (deleted bool, err error) {
	var ok bool
	var count int

	p.countsLock.Lock()
	defer p.countsLock.Unlock()
	if count, ok = p.counts[id]; !ok {
		return false, fmt.Errorf("cannot decrement reference counter: not found")
	}
	count--
	if count == 0 {
		delete(p.counts, id)
		deleted = true
	} else {
		p.counts[id] = count
	}

	return deleted, nil
}
