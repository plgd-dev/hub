package aggregate

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func (e *Published) Version() uint64          { return e.EventVersion }
func (e *Published) EventType() string        { return "ocf.cloud.resourceaggregate.pb.Published" }
func (e *Published) Marshal() ([]byte, error) { return proto.Marshal(e) }
func (e *Published) Unmarshal(b []byte) error { return proto.Unmarshal(b, e) }
func (e *Published) AggregateID() string      { return e.DeviceId + e.Href }
func (e *Published) GroupID() string          { return e.DeviceId }
func (e *Published) IsSnapshot() bool         { return false }
func (e *Published) Timestamp() time.Time     { return time.Unix(0, e.EventTimestamp) }

func (e *Unpublished) Version() uint64          { return e.EventVersion }
func (e *Unpublished) EventType() string        { return "ocf.cloud.resourceaggregate.pb.Unpublished" }
func (e *Unpublished) Marshal() ([]byte, error) { return proto.Marshal(e) }
func (e *Unpublished) Unmarshal(b []byte) error { return proto.Unmarshal(b, e) }
func (e *Unpublished) AggregateID() string      { return e.DeviceId + e.Href }
func (e *Unpublished) GroupID() string          { return e.DeviceId }
func (e *Unpublished) IsSnapshot() bool         { return false }
func (e *Unpublished) Timestamp() time.Time     { return time.Unix(0, e.EventTimestamp) }

func (e *Snapshot) Version() uint64          { return e.EventVersion }
func (e *Snapshot) EventType() string        { return "ocf.cloud.resourceaggregate.pb.Snapshot" }
func (e *Snapshot) Marshal() ([]byte, error) { return proto.Marshal(e) }
func (e *Snapshot) Unmarshal(b []byte) error { return proto.Unmarshal(b, e) }
func (e *Snapshot) AggregateID() string      { return e.DeviceId + e.Href }
func (e *Snapshot) GroupId() string          { return e.DeviceId }
func (e *Snapshot) GroupID() string          { return e.DeviceId }
func (e *Snapshot) IsSnapshot() bool         { return false }
func (e *Snapshot) Timestamp() time.Time     { return time.Unix(0, e.EventTimestamp) }

func (e *Snapshot) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}

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
	}
	return nil
}

func (e *Snapshot) HandleCommand(ctx context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error) {
	switch req := cmd.(type) {
	case *Publish:
		return []eventstore.Event{&Published{DeviceId: req.DeviceId, Href: req.Href, EventVersion: newVersion}}, nil
	case *Unpublish:
		if !e.IsPublished {
			return nil, fmt.Errorf("not allowed to unpublish twice in tests")
		}
		return []eventstore.Event{&Unpublished{DeviceId: req.DeviceId, Href: req.Href, EventVersion: newVersion}}, nil
	}
	return nil, fmt.Errorf("unknown command %T", cmd)
}

func (e *Snapshot) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	e.EventVersion = version
	return e, true
}

type mockEventHandler struct {
	pb []eventstore.EventUnmarshaler
}

func (eh *mockEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		eh.pb = append(eh.pb, eu)
	}
	return nil
}

func testNewEventstore(ctx context.Context, t *testing.T) *mongodb.EventStore {
	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)
	store, err := mongodb.New(
		ctx,
		config.MakeEventsStoreMongoDBConfig(),
		logger,
	)
	require.NoError(t, err)
	require.NotNil(t, store)

	return store
}

func TestAggregate(t *testing.T) {
	ctx := context.Background()
	store := testNewEventstore(ctx, t)
	defer store.Close(ctx)
	defer func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	type Resource struct {
		DeviceID string
		Href     string
	}

	res1 := Resource{
		DeviceID: "1",
		Href:     "ID0",
	}

	res2 := Resource{
		DeviceID: "1",
		Href:     "ID1",
	}

	commandPub1 := Publish{
		DeviceId: res1.DeviceID,
		Href:     res1.Href,
	}

	commandUnpub1 := Unpublish{
		DeviceId: res1.DeviceID,
		Href:     res1.Href,
	}

	commandPub2 := Publish{
		DeviceId: res2.DeviceID,
		Href:     res2.Href,
	}

	commandUnpub2 := Unpublish{
		DeviceId: res2.DeviceID,
		Href:     res2.Href,
	}

	newAggregate := func(deviceID, href string) *Aggregate {
		a, err := NewAggregate(deviceID, deviceID+href, NewDefaultRetryFunc(1), 2, store, func(context.Context) (AggregateModel, error) {
			return &Snapshot{DeviceId: deviceID, Href: href, IsPublished: true}, nil
		}, nil)
		require.NoError(t, err)
		return a
	}

	a := newAggregate(commandPub1.GetDeviceId(), commandPub1.GetHref())
	ev, err := a.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, ev)

	b := newAggregate(commandPub1.GetDeviceId(), commandPub1.GetHref())
	ev, err = b.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, ev)

	c := newAggregate(commandUnpub1.GetDeviceId(), commandUnpub1.GetHref())
	ev, err = c.HandleCommand(ctx, &commandUnpub1)
	require.NoError(t, err)
	require.NotNil(t, ev)

	d := newAggregate(commandUnpub1.GetDeviceId(), commandUnpub1.GetHref())
	ev, err = d.HandleCommand(ctx, &commandUnpub1)
	require.Error(t, err)
	require.Nil(t, ev)

	e := newAggregate(commandPub2.GetDeviceId(), commandPub2.GetHref())
	ev, err = e.HandleCommand(ctx, &commandPub2)
	require.NoError(t, err)
	require.NotNil(t, ev)

	f := newAggregate(commandUnpub2.GetDeviceId(), commandUnpub2.GetHref())
	ev, err = f.HandleCommand(ctx, &commandUnpub2)
	require.NoError(t, err)
	require.NotNil(t, ev)

	g := newAggregate(commandPub1.GetDeviceId(), commandPub1.GetHref())
	ev, err = g.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, ev)

	h := newAggregate(commandUnpub1.GetDeviceId(), commandUnpub1.GetHref())
	ev, err = h.HandleCommand(ctx, &commandUnpub1)
	require.NoError(t, err)
	require.NotNil(t, ev)

	handler := &mockEventHandler{}
	p := eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return handler, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     res1.DeviceID,
			AggregateID: res1.DeviceID + res1.Href,
		},
	})
	require.NoError(t, err)

	concurrencyExcepTestA := newAggregate(commandPub1.GetDeviceId(), commandPub1.GetHref())
	model, err := concurrencyExcepTestA.factoryModel(ctx)
	require.NoError(t, err)

	amodel, err := newAggrModel(ctx, a.groupID, a.aggregateID, a.store, a.LogDebugfFunc, model)
	require.NoError(t, err)

	ev, concurrencyException, err := a.handleCommandWithAggrModel(ctx, &commandPub1, amodel)
	require.NoError(t, err)
	require.False(t, concurrencyException)
	require.NotNil(t, ev)

	ev, concurrencyException, err = a.handleCommandWithAggrModel(ctx, &commandPub1, amodel)
	require.NoError(t, err)
	require.True(t, concurrencyException)
	require.Nil(t, ev)
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func Test_handleRetry(t *testing.T) {
	type args struct {
		ctx       context.Context
		retryFunc RetryFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				ctx:       context.Background(),
				retryFunc: func() (time.Time, error) { return time.Now(), nil },
			},
			wantErr: false,
		},
		{
			name: "err",
			args: args{
				ctx:       context.Background(),
				retryFunc: func() (time.Time, error) { return time.Now().Add(time.Second), errors.New("error") },
			},
			wantErr: true,
		},
		{
			name: "canceled",
			args: args{
				ctx:       canceledContext(),
				retryFunc: func() (time.Time, error) { return time.Now().Add(time.Second), nil },
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleRetry(tt.args.ctx, tt.args.retryFunc); (err != nil) != tt.wantErr {
				t.Errorf("handleRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
