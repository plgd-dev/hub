package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
)

// ErrFunc used by handler to report error from observation
type ErrFunc func(err error)

// GoroutinePoolGoFunc processes actions via provided function
type GoroutinePoolGoFunc func(func()) error

type Event struct {
	Id             string
	EventType      events.EventType
	DeviceID       string
	Href           string
	Representation interface{}
}

//Iter provides iterator over events from eventstore or eventbus.
type Iter interface {
	Next(ctx context.Context, event *Event) bool
	Err() error
}

type Handler interface {
	Handle(ctx context.Context, iter Iter) (err error)
}

// GoroutinePoolHandler submit events to goroutine pool for process them.
type GoroutinePoolHandler struct {
	lock                     sync.Mutex
	aggregateEventsContainer map[string]*eventsProcessor

	goroutinePoolGo GoroutinePoolGoFunc
	eventsHandler   Handler
	errFunc         ErrFunc
}

// NewGoroutinePoolHandler creates new event processor.
func NewGoroutinePoolHandler(
	goroutinePoolGo GoroutinePoolGoFunc,
	eventsHandler Handler,
	errFunc ErrFunc) *GoroutinePoolHandler {

	return &GoroutinePoolHandler{
		goroutinePoolGo:          goroutinePoolGo,
		eventsHandler:            eventsHandler,
		errFunc:                  errFunc,
		aggregateEventsContainer: make(map[string]*eventsProcessor),
	}
}

func (ep *GoroutinePoolHandler) run(ctx context.Context, p *eventsProcessor) error {
	if ep.goroutinePoolGo == nil {
		err := p.process(ctx, ep.eventsHandler)
		ep.tryToDelete(p.name)
		return err
	}
	err := ep.goroutinePoolGo(func() {
		err := p.process(ctx, ep.eventsHandler)
		if err != nil {
			ep.errFunc(err)
		}
		ep.tryToDelete(p.name)
	})
	if err != nil {
		return fmt.Errorf("cannot execute goroutine pool go function: %w", err)
	}
	return nil
}

// Handle pushes event to queue and process the queue by goroutine pool.
func (ep *GoroutinePoolHandler) Handle(ctx context.Context, event Event) (err error) {
	ed := ep.getEventsData(event.Id)
	spawnGo := ed.push([]Event{event})
	if spawnGo {
		err := ep.run(ctx, ed)
		if err != nil {
			return fmt.Errorf("cannot handle events: %w", err)
		}
	}
	return nil
}

func (ep *GoroutinePoolHandler) getEventsData(name string) *eventsProcessor {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	ed, ok := ep.aggregateEventsContainer[name]
	if !ok {
		ed = newEventsProcessor(name)
		ep.aggregateEventsContainer[name] = ed
	}
	return ed
}

func (ep *GoroutinePoolHandler) tryToDelete(name string) {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	ed, ok := ep.aggregateEventsContainer[name]
	if ok {
		ed.lock.Lock()
		defer ed.lock.Unlock()
		if !ed.isProcessed {
			delete(ep.aggregateEventsContainer, name)
		}
	}
}

type eventsProcessor struct {
	name        string
	queue       []Event
	isProcessed bool
	lock        sync.Mutex
}

func newEventsProcessor(name string) *eventsProcessor {
	return &eventsProcessor{
		name:  name,
		queue: make([]Event, 0, 16),
	}
}

func (ed *eventsProcessor) push(events []Event) bool {
	ed.lock.Lock()
	defer ed.lock.Unlock()
	ed.queue = append(ed.queue, events...)
	if ed.isProcessed == false {
		ed.isProcessed = true
		return true
	}
	return false
}

func (ed *eventsProcessor) pop() []Event {
	ed.lock.Lock()
	defer ed.lock.Unlock()
	if len(ed.queue) > 0 {
		res := ed.queue
		ed.queue = make([]Event, 0, 16)
		return res
	}
	ed.isProcessed = false
	return nil
}

func (ed *eventsProcessor) process(ctx context.Context, eh Handler) error {
	for {
		events := ed.pop()
		if len(events) == 0 {
			return nil
		}

		i := iter{
			events: events,
		}

		if err := eh.Handle(ctx, &i); err != nil {
			ed.lock.Lock()
			defer ed.lock.Unlock()
			ed.isProcessed = false
			return fmt.Errorf("cannot process event: %w", err)
		}
	}
}

type iter struct {
	events []Event
	idx    int
}

func (i *iter) Next(ctx context.Context, e *Event) bool {
	if i.idx >= len(i.events) {
		return false
	}
	*e = i.events[i.idx]
	i.idx++
	return true
}

func (i *iter) Err() error {
	return nil
}
