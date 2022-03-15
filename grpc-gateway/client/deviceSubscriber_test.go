package client

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
)

type testOperations struct {
}

func (o testOperations) RetrieveResource(ctx context.Context, event *events.ResourceRetrievePending) error {
	return nil
}
func (o testOperations) UpdateResource(ctx context.Context, event *events.ResourceUpdatePending) error {
	return nil
}
func (o testOperations) DeleteResource(ctx context.Context, event *events.ResourceDeletePending) error {
	return nil
}
func (o testOperations) CreateResource(ctx context.Context, event *events.ResourceCreatePending) error {
	return nil
}
func (o testOperations) UpdateDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	return nil
}
func (o testOperations) OnDeviceSubscriberReconnectError(err error) {}

func TestDeviceSubscriptionHandlersWantToProcessEvent(t *testing.T) {
	type args struct {
		key          uint64
		eventVersion uint64
		fromDB       bool
	}
	tests := []struct {
		name    string
		preArgs []args
		args    args
		want    bool
	}{
		{
			name: "coldStart",
			args: args{
				key:          1,
				eventVersion: 0,
				fromDB:       true,
			},
			want: true,
		},
		{
			name: "duplicitUpdateDBNats",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 0,
					fromDB:       true,
				},
			},
			args: args{
				key:          1,
				eventVersion: 0,
				fromDB:       false,
			},
			want: false,
		},
		{
			name: "duplicitUpdateNatsDB",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 0,
					fromDB:       false,
				},
			},
			args: args{
				key:          1,
				eventVersion: 0,
				fromDB:       true,
			},
			want: false,
		},
		{
			name: "db+reorderNats",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 0,
					fromDB:       true,
				},
				{
					key:          1,
					eventVersion: 2,
					fromDB:       false,
				},
			},
			args: args{
				key:          1,
				eventVersion: 1,
				fromDB:       false,
			},
			want: true,
		},
		{
			name: "duplicitDB",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 0,
					fromDB:       true,
				},
			},
			args: args{
				key:          1,
				eventVersion: 0,
				fromDB:       true,
			},
			want: false,
		},
		{
			name: "reorderDB",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 1,
					fromDB:       true,
				},
			},
			args: args{
				key:          1,
				eventVersion: 0,
				fromDB:       true,
			},
			want: false,
		},
		{
			name: "natsDB",
			preArgs: []args{
				{
					key:          1,
					eventVersion: 0,
					fromDB:       false,
				},
			},
			args: args{
				key:          1,
				eventVersion: 1,
				fromDB:       true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeviceSubscriptionHandlers(testOperations{})
			for _, a := range tt.preArgs {
				h.wantToProcessEvent(a.key, a.eventVersion, a.fromDB)
			}
			got := h.wantToProcessEvent(tt.args.key, tt.args.eventVersion, tt.args.fromDB)
			require.Equal(t, tt.want, got)
		})
	}
}
