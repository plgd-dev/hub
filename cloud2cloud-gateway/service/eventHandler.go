package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	netHttp "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	oapiStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

type EventHandler struct {
	store           store.Store
	goroutinePoolGo GoroutinePoolGoFunc
}

func newEventHandler(
	store store.Store,
	goroutinePoolGo GoroutinePoolGoFunc,
) *EventHandler {
	return &EventHandler{
		store:           store,
		goroutinePoolGo: goroutinePoolGo,
	}
}

type incrementSubscriptionSequenceNumberFunc func(ctx context.Context, subscriptionID string) (uint64, error)

func emitEvent(ctx context.Context, eventType events.EventType, s store.Subscription, incrementSubscriptionSequenceNumber incrementSubscriptionSequenceNumberFunc, rep interface{}) (remove bool, err error) {
	log.Debugf("emitEvent: %v: %+v", eventType, s)
	defer log.Debugf("emitEvent done: %v: %+v", eventType, s)
	client := netHttp.Client{}
	encoder, err := getEncoder(s.ContentType)
	if err != nil {
		return false, fmt.Errorf("cannot get encoder: %w", err)
	}
	seqNum, err := incrementSubscriptionSequenceNumber(ctx, s.ID)
	if err != nil {
		return false, fmt.Errorf("cannot increment sequence number: %w", err)
	}

	r, w := io.Pipe()

	req, err := netHttp.NewRequest("POST", s.URL, r)
	if err != nil {
		return false, fmt.Errorf("cannot create post request: %w", err)
	}
	timestamp := time.Now()
	req.Header.Set(events.EventTypeKey, string(eventType))
	req.Header.Set(events.SubscriptionIDKey, s.ID)
	req.Header.Set(events.SequenceNumberKey, strconv.FormatUint(seqNum, 10))
	req.Header.Set(events.CorrelationIDKey, s.CorrelationID)
	req.Header.Set(events.EventTimestampKey, strconv.FormatInt(timestamp.Unix(), 10))
	var body []byte
	if rep != nil {
		body, err = encoder(rep)
		if err != nil {
			return false, fmt.Errorf("cannot encode data to body: %w", err)
		}
		req.Header.Set(events.ContentTypeKey, s.ContentType)
		go func() {
			defer w.Close()
			if len(body) > 0 {
				_, err := w.Write(body)
				if err != nil {
					log.Errorf("cannot write data to client: %v", err)
				}
			}
		}()
	} else {
		w.Close()
	}
	req.Header.Set(events.EventSignatureKey, events.CalculateEventSignature(
		s.SigningSecret,
		req.Header.Get(events.ContentTypeKey),
		eventType,
		req.Header.Get(events.SubscriptionIDKey),
		seqNum,
		timestamp,
		body,
	))

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("cannot post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != netHttp.StatusOK {
		errBody, _ := ioutil.ReadAll(resp.Body)
		return resp.StatusCode == netHttp.StatusGone, fmt.Errorf("%v: unexpected statusCode %v: body: '%v'", s.URL, resp.StatusCode, string(errBody))
	}
	return eventType == events.EventType_SubscriptionCanceled, nil
}

type resourceSubscriptionHandler struct {
	store           store.Store
	goroutinePoolGo GoroutinePoolGoFunc
	event           Event
}

func newResourceSubscriptionHandler(
	store store.Store,
	goroutinePoolGo GoroutinePoolGoFunc,
	event Event,
) *resourceSubscriptionHandler {
	return &resourceSubscriptionHandler{
		store:           store,
		goroutinePoolGo: goroutinePoolGo,
		event:           event,
	}
}

func (c *resourceSubscriptionHandler) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	var wg sync.WaitGroup
	for {
		var s store.Subscription
		if !iter.Next(ctx, &s) {
			break
		}
		if s.SequenceNumber == 0 {
			// first event is emitted by after create subscription.
			continue
		}
		for _, e := range s.EventTypes {
			if e == c.event.EventType {
				wg.Add(1)
				c.goroutinePoolGo(func() {
					defer wg.Done()
					rs := s
					remove, err := emitEvent(ctx, c.event.EventType, rs, c.store.IncrementSubscriptionSequenceNumber, c.event.Representation)
					if remove {
						c.store.PopSubscription(ctx, rs.ID)
					}
					if err != nil {
						log.Errorf("cannot emit event: %v", err)
					}
				})
				break
			}
		}
	}
	wg.Wait()
	return iter.Err()
}

func (h *EventHandler) processResourceEvent(e Event) error {
	err := h.store.LoadSubscriptions(
		context.Background(),
		store.SubscriptionQuery{
			Type:     oapiStore.Type_Resource,
			DeviceID: e.DeviceID,
			Href:     e.Href,
		},
		newResourceSubscriptionHandler(h.store, h.goroutinePoolGo, e),
	)
	if err != nil {
		return fmt.Errorf("cannot process resource event (DeviceID: %v, Href: %v): %w", e.DeviceID, e.Href, err)
	}
	return nil
}

type deviceSubscriptionHandlerEvent struct {
	store           store.Store
	goroutinePoolGo GoroutinePoolGoFunc
	event           Event
}

func newDeviceSubscriptionHandler(
	store store.Store,
	goroutinePoolGo GoroutinePoolGoFunc,
	event Event,
) *deviceSubscriptionHandlerEvent {
	return &deviceSubscriptionHandlerEvent{
		store:           store,
		goroutinePoolGo: goroutinePoolGo,
		event:           event,
	}
}

func makeLinksRepresentation(eventType events.EventType, models []eventstore.Model) []schema.ResourceLink {
	result := make([]schema.ResourceLink, 0, len(models))
	for _, m := range models {
		c := m.(*resourceCtx).Clone()
		switch eventType {
		case events.EventType_ResourcesPublished:
			if c.resource.GetHref() == cloud.StatusHref {
				continue
			}
			if c.isPublished {
				result = append(result, makeResourceLink(c.resource))
			}
		case events.EventType_ResourcesUnpublished:
			if c.resource.GetHref() == cloud.StatusHref {
				continue
			}
			if !c.isPublished {
				result = append(result, makeResourceLink(c.resource))
			}
		}
	}
	return result
}

func (c *deviceSubscriptionHandlerEvent) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	var wg sync.WaitGroup
	for {
		var s store.Subscription
		if !iter.Next(ctx, &s) {
			break
		}
		for _, e := range s.EventTypes {
			if e == c.event.EventType {
				wg.Add(1)
				err := c.goroutinePoolGo(func() {
					defer wg.Done()
					rs := s
					remove, err := emitEvent(ctx, c.event.EventType, rs, c.store.IncrementSubscriptionSequenceNumber, c.event.Representation)
					if remove {
						c.store.PopSubscription(ctx, rs.ID)
					}
					if err != nil {
						log.Errorf("cannot emit event: %v", err)
					}
				})
				if err != nil {
					wg.Done()
				}
				break
			}
		}
	}
	wg.Wait()
	return iter.Err()
}

func (h *EventHandler) processDeviceEvent(e Event) error {
	err := h.store.LoadSubscriptions(
		context.Background(),
		store.SubscriptionQuery{
			Type:     oapiStore.Type_Device,
			DeviceID: e.DeviceID,
		},
		newDeviceSubscriptionHandler(h.store, h.goroutinePoolGo, e),
	)
	if err != nil {
		return fmt.Errorf("cannot process resource event (DeviceID: %v, Href: %v): %w", e.DeviceID, e.Href, err)
	}
	return nil
}

func (h *EventHandler) processEvent(e Event) error {
	switch e.EventType {
	case events.EventType_ResourceChanged:
		return h.processResourceEvent(e)
	case events.EventType_ResourcesPublished, events.EventType_ResourcesUnpublished:
		return h.processDeviceEvent(e)
	}
	return fmt.Errorf("cannot process event: unknown event-type %v", e.EventType)
}

func (h *EventHandler) Handle(ctx context.Context, iter Iter) (err error) {
	for {
		var e Event
		if !iter.Next(ctx, &e) {
			break
		}
		err := h.processEvent(e)
		if err != nil {
			log.Errorf("%v", err)
		}
	}

	return iter.Err()

}
