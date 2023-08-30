package subscription

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	eventbusPb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	"google.golang.org/protobuf/proto"
)

type (
	ErrFunc               = func(err error)
	SendEventWithTypeFunc = func(e *pb.Event, typeBit FilterBitmask) error
)

type eventSubject struct {
	senders      map[uint64]SendEventWithTypeFunc
	subscription *nats.Subscription
	sync.Mutex
}

func newEventSubject() *eventSubject {
	return &eventSubject{
		senders: make(map[uint64]SendEventWithTypeFunc),
	}
}

func registrationEventToGrpcEvent(e *isEvents.Event) (*pb.Event, FilterBitmask) {
	switch {
	case e.GetDevicesRegistered() != nil:
		return &pb.Event{
			Type: &pb.Event_DeviceRegistered_{
				DeviceRegistered: &pb.Event_DeviceRegistered{
					DeviceIds:            e.GetDevicesRegistered().GetDeviceIds(),
					OpenTelemetryCarrier: e.GetDevicesRegistered().GetOpenTelemetryCarrier(),
					EventMetadata:        e.GetDevicesRegistered().GetEventMetadata(),
				},
			},
		}, FilterBitmaskDeviceRegistered
	case e.GetDevicesUnregistered() != nil:
		return &pb.Event{
			Type: &pb.Event_DeviceUnregistered_{
				DeviceUnregistered: &pb.Event_DeviceUnregistered{
					DeviceIds:            e.GetDevicesUnregistered().GetDeviceIds(),
					OpenTelemetryCarrier: e.GetDevicesUnregistered().GetOpenTelemetryCarrier(),
					EventMetadata:        e.GetDevicesUnregistered().GetEventMetadata(),
				},
			},
		}, FilterBitmaskDeviceUnregistered
	}
	return nil, 0
}

func sendToSenders(ev *pb.Event, bit FilterBitmask, senders []func(e *pb.Event, typeBit FilterBitmask) error) error {
	var errors *multierror.Error
	for _, send := range senders {
		err := send(ev, bit)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func (d *eventSubject) handleRegistrationsEvent(msg *nats.Msg) error {
	var e isEvents.Event
	if err := utils.Unmarshal(msg.Data, &e); err != nil {
		return err
	}
	ev, bit := registrationEventToGrpcEvent(&e)
	if ev == nil {
		return fmt.Errorf("unhandled registrations event type('%T')", e.GetType())
	}
	return sendToSenders(ev, bit, d.copySenders())
}

type devicesEventToGrpcFunc func(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error)

const errorUnmarshalEventFmt = "failed to unmarshal event %v: %w"

func convResourcesPublished(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceLinksPublished
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourcePublished{
			ResourcePublished: &e,
		},
	}, FilterBitmaskResourcesPublished, nil
}

func convResourcesUnpublished(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceLinksUnpublished
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUnpublished{
			ResourceUnpublished: &e,
		},
	}, FilterBitmaskResourcesUnpublished, nil
}

func convResourceChanged(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceChanged
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: &e,
		},
	}, FilterBitmaskResourceChanged, nil
}

func convResourceUpdatePending(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceUpdatePending
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: &e,
		},
	}, FilterBitmaskResourceUpdatePending, nil
}

func convResourceUpdated(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceUpdated
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUpdated{
			ResourceUpdated: &e,
		},
	}, FilterBitmaskResourceUpdated, nil
}

func convResourceRetrievePending(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceRetrievePending
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: &e,
		},
	}, FilterBitmaskResourceRetrievePending, nil
}

func convResourceRetrieved(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceRetrieved
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: &e,
		},
	}, FilterBitmaskResourceRetrieved, nil
}

func convResourceDeletePending(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceDeletePending
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceDeletePending{
			ResourceDeletePending: &e,
		},
	}, FilterBitmaskResourceDeletePending, nil
}

func convResourceDeleted(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceDeleted
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceDeleted{
			ResourceDeleted: &e,
		},
	}, FilterBitmaskResourceDeleted, nil
}

func convResourceCreatePending(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceCreatePending
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: &e,
		},
	}, FilterBitmaskResourceCreatePending, nil
}

func convResourceCreated(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.ResourceCreated
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: &e,
		},
	}, FilterBitmaskResourceCreated, nil
}

func convDeviceMetadataUpdatePending(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.DeviceMetadataUpdatePending
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_DeviceMetadataUpdatePending{
			DeviceMetadataUpdatePending: &e,
		},
	}, FilterBitmaskDeviceMetadataUpdatePending, nil
}

func convDeviceMetadataUpdated(ev *eventbusPb.Event) (*pb.Event, FilterBitmask, error) {
	var e events.DeviceMetadataUpdated
	if err := utils.Unmarshal(ev.GetData(), &e); err != nil {
		return nil, 0, fmt.Errorf(errorUnmarshalEventFmt, ev.GetEventType(), err)
	}
	return &pb.Event{
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &e,
		},
	}, FilterBitmaskDeviceMetadataUpdated, nil
}

var devicesEventToGrpcPb = map[string]devicesEventToGrpcFunc{
	(&events.ResourceCreatePending{}).EventType():       convResourceCreatePending,
	(&events.ResourceCreated{}).EventType():             convResourceCreated,
	(&events.ResourceRetrievePending{}).EventType():     convResourceRetrievePending,
	(&events.ResourceRetrieved{}).EventType():           convResourceRetrieved,
	(&events.ResourceUpdatePending{}).EventType():       convResourceUpdatePending,
	(&events.ResourceUpdated{}).EventType():             convResourceUpdated,
	(&events.ResourceDeletePending{}).EventType():       convResourceDeletePending,
	(&events.ResourceDeleted{}).EventType():             convResourceDeleted,
	(&events.DeviceMetadataUpdatePending{}).EventType(): convDeviceMetadataUpdatePending,
	(&events.DeviceMetadataUpdated{}).EventType():       convDeviceMetadataUpdated,
	(&events.ResourceChanged{}).EventType():             convResourceChanged,
	(&events.ResourceLinksPublished{}).EventType():      convResourcesPublished,
	(&events.ResourceLinksUnpublished{}).EventType():    convResourcesUnpublished,
}

func (d *eventSubject) copySenders() []SendEventWithTypeFunc {
	d.Lock()
	defer d.Unlock()
	senders := make([]SendEventWithTypeFunc, 0, len(d.senders))
	for _, h := range d.senders {
		senders = append(senders, h)
	}
	return senders
}

func (d *eventSubject) handleDevicesEvent(msg *nats.Msg) error {
	var e eventbusPb.Event
	if err := proto.Unmarshal(msg.Data, &e); err != nil {
		return err
	}

	h, ok := devicesEventToGrpcPb[e.GetEventType()]
	if !ok {
		return fmt.Errorf("unhandled event type('%v')", e.GetEventType())
	}
	ev, bit, err := h(&e)
	if err != nil {
		return err
	}
	return sendToSenders(ev, bit, d.copySenders())
}

func (d *eventSubject) handleEvent(msg *nats.Msg) error {
	if strings.Contains(msg.Subject, "."+isEvents.Registrations) {
		return d.handleRegistrationsEvent(msg)
	} else if strings.Contains(msg.Subject, "."+utils.Devices) {
		return d.handleDevicesEvent(msg)
	}
	return fmt.Errorf("cannot process event from unknown subject(%v)", msg.Subject)
}

func (d *eventSubject) AddHandlerLocked(id uint64, h SendEventWithTypeFunc) bool {
	if _, ok := d.senders[id]; !ok {
		d.senders[id] = h
		return true
	}
	return false
}

func (d *eventSubject) RemoveHandlerLocked(v uint64) bool {
	delete(d.senders, v)
	return len(d.senders) == 0
}

func (d *eventSubject) subscribeLocked(subject string, subscribe func(subj string, cb nats.MsgHandler) (*nats.Subscription, error), handle func(msg *nats.Msg)) error {
	if d.subscription == nil {
		sub, err := subscribe(subject, handle)
		if err != nil {
			return err
		}
		d.subscription = sub
	}
	return nil
}

type SubscriptionsCache struct {
	subjects  *kitSync.Map
	conn      *nats.Conn
	errFunc   ErrFunc
	handlerID uint64
}

func NewSubscriptionsCache(conn *nats.Conn, errFunc ErrFunc) *SubscriptionsCache {
	c := &SubscriptionsCache{
		subjects: kitSync.NewMap(),
		conn:     conn,
		errFunc:  errFunc,
	}
	return c
}

func (c *SubscriptionsCache) makeCloseFunc(subject string, id uint64) func() {
	return func() {
		var sub *nats.Subscription
		c.subjects.ReplaceWithFunc(subject, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, doDelete bool) {
			if !oldLoaded {
				return nil, true
			}
			s := oldValue.(*eventSubject)
			s.Lock()
			defer s.Unlock()
			if s.RemoveHandlerLocked(id) {
				sub = s.subscription
				return nil, true
			}
			return s, false
		})
		if sub != nil {
			err := sub.Unsubscribe()
			if err != nil {
				c.errFunc(fmt.Errorf("cannot unsubscribe from subject('%v'): %w", subject, err))
			}
		}
	}
}

// Create or get owner subject, lock it, execute function and unlock it
func (c *SubscriptionsCache) executeOnLockedeventSubject(subject string, fn func(*eventSubject) error) error {
	val, _ := c.subjects.LoadOrStoreWithFunc(subject, func(value interface{}) interface{} {
		v := value.(*eventSubject)
		v.Lock()
		return v
	}, func() interface{} {
		v := newEventSubject()
		v.Lock()
		return v
	})
	s := val.(*eventSubject)
	defer s.Unlock()
	return fn(s)
}

// Subscribe register onEvents handler and creates a NATS subscription, if it does not exist.
// To free subscription call the returned close function.
func (c *SubscriptionsCache) Subscribe(subject string, onEvent SendEventWithTypeFunc) (closeFn func(), err error) {
	closeFunc := func() {
		// Do nothing if no owner subject is found
	}
	err = c.executeOnLockedeventSubject(subject, func(s *eventSubject) error {
		if onEvent != nil {
			handlerID := atomic.AddUint64(&c.handlerID, 1)
			for !s.AddHandlerLocked(handlerID, onEvent) {
				handlerID = atomic.AddUint64(&c.handlerID, 1)
			}
			closeFunc = c.makeCloseFunc(subject, handlerID)
		}
		if s.subscription == nil {
			err := s.subscribeLocked(subject, c.conn.Subscribe, func(msg *nats.Msg) {
				if err := s.handleEvent(msg); err != nil {
					c.errFunc(err)
				}
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		closeFunc()
		return nil, err
	}

	return closeFunc, nil
}
