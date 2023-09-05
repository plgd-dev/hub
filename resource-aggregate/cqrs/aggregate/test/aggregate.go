package test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

type Command = interface{}

func (e *Published) Version() uint64            { return e.EventVersion }
func (e *Published) EventType() string          { return "published" }
func (e *Published) Marshal() ([]byte, error)   { return proto.Marshal(e) }
func (e *Published) Unmarshal(b []byte) error   { return proto.Unmarshal(b, e) }
func (e *Published) AggregateID() string        { return e.DeviceId + e.Href }
func (e *Published) GroupID() string            { return e.DeviceId }
func (e *Published) IsSnapshot() bool           { return false }
func (e *Published) Timestamp() time.Time       { return time.Unix(0, e.EventTimestamp) }
func (e *Published) ETag() *eventstore.ETagData { return nil }

func (e *Unpublished) Version() uint64            { return e.EventVersion }
func (e *Unpublished) EventType() string          { return "unpublished" }
func (e *Unpublished) Marshal() ([]byte, error)   { return proto.Marshal(e) }
func (e *Unpublished) Unmarshal(b []byte) error   { return proto.Unmarshal(b, e) }
func (e *Unpublished) AggregateID() string        { return e.DeviceId + e.Href }
func (e *Unpublished) GroupID() string            { return e.DeviceId }
func (e *Unpublished) IsSnapshot() bool           { return false }
func (e *Unpublished) Timestamp() time.Time       { return time.Unix(0, e.EventTimestamp) }
func (e *Unpublished) ETag() *eventstore.ETagData { return nil }

func (e *Snapshot) Version() uint64            { return e.EventVersion }
func (e *Snapshot) EventType() string          { return "snapshot" }
func (e *Snapshot) Marshal() ([]byte, error)   { return proto.Marshal(e) }
func (e *Snapshot) Unmarshal(b []byte) error   { return proto.Unmarshal(b, e) }
func (e *Snapshot) AggregateID() string        { return e.DeviceId + e.Href }
func (e *Snapshot) GroupId() string            { return e.DeviceId }
func (e *Snapshot) GroupID() string            { return e.DeviceId }
func (e *Snapshot) IsSnapshot() bool           { return true }
func (e *Snapshot) Timestamp() time.Time       { return time.Unix(0, e.EventTimestamp) }
func (e *Snapshot) ETag() *eventstore.ETagData { return nil }

func (e *Snapshot) handleEvent(eu eventstore.EventUnmarshaler) error {
	if eu.EventType() == "" {
		return errors.New("cannot determine type of event")
	}
	switch eu.EventType() {
	case (&Snapshot{}).EventType():
		var s Snapshot
		if err := eu.Unmarshal(&s); err != nil {
			return err
		}
		e.DeviceId = s.GetDeviceId()
		e.Href = s.GetHref()
		e.EventVersion = s.GetEventVersion()
		e.IsPublished = s.GetIsPublished()
	case (&Published{}).EventType():
		var s Published
		if err := eu.Unmarshal(&s); err != nil {
			return err
		}
		e.DeviceId = s.GetDeviceId()
		e.Href = s.GetHref()
		e.EventVersion = s.GetEventVersion()
		e.IsPublished = true
	case (&Unpublished{}).EventType():
		var s Unpublished
		if err := eu.Unmarshal(&s); err != nil {
			return err
		}
		e.DeviceId = s.GetDeviceId()
		e.Href = s.GetHref()
		e.EventVersion = s.GetEventVersion()
		e.IsPublished = false
	}
	return nil
}

func (e *Snapshot) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if err := e.handleEvent(eu); err != nil {
			return err
		}
	}
	return nil
}

func (e *Snapshot) HandleCommand(_ context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error) {
	switch req := cmd.(type) {
	case *Publish:
		e.IsPublished = true
		return []eventstore.Event{&Published{DeviceId: req.DeviceId, Href: req.Href, EventVersion: newVersion}}, nil
	case *Unpublish:
		if !e.IsPublished {
			return nil, fmt.Errorf("not allowed to unpublish twice in tests")
		}
		e.IsPublished = false
		return []eventstore.Event{&Unpublished{DeviceId: req.DeviceId, Href: req.Href, EventVersion: newVersion}}, nil
	}
	return nil, fmt.Errorf("unknown command %T", cmd)
}

func (e *Snapshot) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	e.EventVersion = version
	return e, true
}
