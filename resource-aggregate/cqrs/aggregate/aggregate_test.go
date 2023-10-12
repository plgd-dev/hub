package aggregate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

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

func TestAggregate(t *testing.T) {
	ctx := context.Background()
	store, tearDown := testNewEventstore(ctx, t)
	defer func() {
		tearDown()
	}()

	type Resource struct {
		DeviceID string
		Href     string
	}

	res1 := Resource{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Href:     "ID0",
	}

	res2 := Resource{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Href:     "ID1",
	}

	commandPub1 := raTest.Publish{
		DeviceId: res1.DeviceID,
		Href:     res1.Href,
	}

	commandUnpub1 := raTest.Unpublish{
		DeviceId: res1.DeviceID,
		Href:     res1.Href,
	}

	commandPub2 := raTest.Publish{
		DeviceId: res2.DeviceID,
		Href:     res2.Href,
	}

	commandUnpub2 := raTest.Unpublish{
		DeviceId: res2.DeviceID,
		Href:     res2.Href,
	}

	newAggregate := func(deviceID, href string) *aggregate.Aggregate {
		a, err := aggregate.NewAggregate(deviceID, commands.NewResourceID(deviceID, href).ToUUID().String(), aggregate.NewDefaultRetryFunc(1), store, func(context.Context) (aggregate.AggregateModel, error) {
			return &raTest.Snapshot{DeviceId: deviceID, Href: href, IsPublished: true}, nil
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
			AggregateID: commands.NewResourceID(res1.DeviceID, res1.Href).ToUUID().String(),
		},
	})
	require.NoError(t, err)

	concurrencyExcepTestA := newAggregate(commandPub1.GetDeviceId(), commandPub1.GetHref())
	model, err := concurrencyExcepTestA.FactoryModel(ctx)
	require.NoError(t, err)

	amodel, err := aggregate.NewAggregateModel(ctx, a.GroupID(), a.AggregateID(), store, a.LogDebugfFunc, model)
	require.NoError(t, err)

	ev, concurrencyException, err := a.HandleCommandWithAggregateModelWrapper(ctx, &commandPub1, amodel)
	require.NoError(t, err)
	require.False(t, concurrencyException)
	require.NotNil(t, ev)

	ev, concurrencyException, err = a.HandleCommandWithAggregateModelWrapper(ctx, &commandPub1, amodel)
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
		retryFunc aggregate.RetryFunc
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
			if err := aggregate.HandleRetry(tt.args.ctx, tt.args.retryFunc); (err != nil) != tt.wantErr {
				t.Errorf("handleRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
