package mongodb

import (
	"context"
	"testing"

	"github.com/go-ocf/cloud/openapi-connector/store"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_FindOrCreateSubscription(t *testing.T) {
	type args struct {
		sub store.Subscription
	}
	tests := []struct {
		name    string
		args    args
		want    store.Subscription
		wantErr bool
	}{
		{
			name: "Type_Devices - valid",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "0",
					Type:            store.Type_Devices,
					LinkedAccountID: "testLinkedAccountID",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "0",
				Type:            store.Type_Devices,
				LinkedAccountID: "testLinkedAccountID",
			},
		},
		{
			name: "Type_Devices - valid - duplicit with same LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "0",
					Type:            store.Type_Devices,
					LinkedAccountID: "testLinkedAccountID",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "0",
				Type:            store.Type_Devices,
				LinkedAccountID: "testLinkedAccountID",
			},
		},
		{
			name: "Type_Devices - error - duplicit with different LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "1",
					Type:            store.Type_Devices,
					LinkedAccountID: "testLinkedAccountID",
				},
			},
			wantErr: true,
		},
		{
			name: "Type_Device - valid",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "1",
					Type:            store.Type_Device,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "1",
				Type:            store.Type_Device,
				LinkedAccountID: "testLinkedAccountID",
				DeviceID:        "testDeviceID",
			},
		},

		{
			name: "Type_Device - valid - duplicit with same LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "1",
					Type:            store.Type_Device,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "1",
				Type:            store.Type_Device,
				LinkedAccountID: "testLinkedAccountID",
				DeviceID:        "testDeviceID",
			},
		},
		{
			name: "Type_Device - error - duplicit with different LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "2",
					Type:            store.Type_Device,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
				},
			},
			wantErr: true,
		},
		{
			name: "Type_Resource - valid",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "2",
					Type:            store.Type_Resource,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
					Href:            "testHref",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "2",
				Type:            store.Type_Resource,
				LinkedAccountID: "testLinkedAccountID",
				DeviceID:        "testDeviceID",
				Href:            "testHref",
			},
		},

		{
			name: "Type_Resource - valid - duplicit with same LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "2",
					Type:            store.Type_Resource,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
					Href:            "testHref",
				},
			},
			want: store.Subscription{
				SubscriptionID:  "2",
				Type:            store.Type_Resource,
				LinkedAccountID: "testLinkedAccountID",
				DeviceID:        "testDeviceID",
				Href:            "testHref",
			},
		},
		{
			name: "Type_Resource - error - duplicit with different LinkedAccountID",
			args: args{
				sub: store.Subscription{
					SubscriptionID:  "3",
					Type:            store.Type_Resource,
					LinkedAccountID: "testLinkedAccountID",
					DeviceID:        "testDeviceID",
					Href:            "testHref",
				},
			},
			wantErr: true,
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	require.NoError(err)
	defer s.Clear(ctx)

	assert := assert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.FindOrCreateSubscription(ctx, tt.args.sub)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.want, got)
			}
		})
	}
}

type testSubscriptionHandler struct {
	subs []store.Subscription
}

func (h *testSubscriptionHandler) Handle(ctx context.Context, iter store.SubscriptionIter) (err error) {
	var sub store.Subscription
	for iter.Next(ctx, &sub) {
		h.subs = append(h.subs, sub)
	}
	return iter.Err()
}

func TestStore_LoadSubscriptions(t *testing.T) {

	subs := []store.Subscription{
		store.Subscription{
			SubscriptionID:  "0",
			Type:            store.Type_Devices,
			LinkedAccountID: "testLinkedAccountID",
		},
		store.Subscription{
			SubscriptionID:  "1",
			Type:            store.Type_Device,
			LinkedAccountID: "testLinkedAccountID",
			DeviceID:        "testDeviceID",
		},
		store.Subscription{
			SubscriptionID:  "2",
			Type:            store.Type_Resource,
			LinkedAccountID: "testLinkedAccountID",
			DeviceID:        "testDeviceID",
			Href:            "testHref",
		},
	}

	type args struct {
		queries []store.SubscriptionQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []store.Subscription
	}{
		{
			name: "bySubscriptionID",
			args: args{
				queries: []store.SubscriptionQuery{store.SubscriptionQuery{SubscriptionID: "2"}},
			},
			want: []store.Subscription{subs[2]},
		},
		{
			name: "byLinkedAccountID",
			args: args{
				queries: []store.SubscriptionQuery{store.SubscriptionQuery{LinkedAccountID: "testLinkedAccountID"}},
			},
			want: subs,
		},
		{
			name: "byResource",
			args: args{
				queries: []store.SubscriptionQuery{store.SubscriptionQuery{Type: store.Type_Resource, DeviceID: "testDeviceID", Href: "testHref"}},
			},
			want: []store.Subscription{subs[2]},
		},
		{
			name: "byDevice",
			args: args{
				queries: []store.SubscriptionQuery{store.SubscriptionQuery{Type: store.Type_Device, DeviceID: "testDeviceID"}},
			},
			want: []store.Subscription{subs[1]},
		},
		{
			name: "allDeviceSubscriptions",
			args: args{
				queries: []store.SubscriptionQuery{store.SubscriptionQuery{Type: store.Type_Device}},
			},
			want: []store.Subscription{subs[1]},
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	require.NoError(err)
	defer s.Clear(ctx)

	assert := assert.New(t)

	for _, sub := range subs {
		_, err := s.FindOrCreateSubscription(ctx, sub)
		require.NoError(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testSubscriptionHandler
			err := s.LoadSubscriptions(ctx, tt.args.queries, &h)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.want, h.subs)
			}
		})
	}
}

func TestStore_RemoveSubscriptions(t *testing.T) {
	subs := []store.Subscription{
		store.Subscription{
			SubscriptionID:  "0",
			Type:            store.Type_Devices,
			LinkedAccountID: "testLinkedAccountID",
		},
		store.Subscription{
			SubscriptionID:  "1",
			Type:            store.Type_Device,
			LinkedAccountID: "testLinkedAccountID",
			DeviceID:        "testDeviceID",
		},
		store.Subscription{
			SubscriptionID:  "2",
			Type:            store.Type_Resource,
			LinkedAccountID: "testLinkedAccountID",
			DeviceID:        "testDeviceID",
			Href:            "testHref",
		},
	}
	type args struct {
		query store.SubscriptionQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "byLinkedAccountID",
			args: args{
				query: store.SubscriptionQuery{
					LinkedAccountID: "testLinkedAccountID",
				},
			},
		},
		{
			name: "byDeviceID",
			args: args{
				query: store.SubscriptionQuery{
					DeviceID: "testDeviceID",
				},
			},
			wantErr: true,
		},
		{
			name: "byHref",
			args: args{
				query: store.SubscriptionQuery{
					Href: "testHref",
				},
			},
			wantErr: true,
		},
		{
			name: "byType",
			args: args{
				query: store.SubscriptionQuery{
					Type: store.Type_Devices,
				},
			},
			wantErr: true,
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	require.NoError(err)
	defer s.Clear(ctx)
	assert := assert.New(t)

	for _, sub := range subs {
		_, err := s.FindOrCreateSubscription(ctx, sub)
		require.NoError(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.RemoveSubscriptions(ctx, tt.args.query)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
