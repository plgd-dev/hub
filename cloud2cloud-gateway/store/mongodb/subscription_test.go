package mongodb

import (
	"context"
	"testing"

	oapiConStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/stretchr/testify/assert"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func newStore(ctx context.Context, t *testing.T, cfg Config) *Store {
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	defer dialCertManager.Close()
	tlsConfig := dialCertManager.GetClientTLSConfig()
	s, err := NewStore(ctx, cfg, WithTLS(tlsConfig))
	require.NoError(t, err)
	return s
}

func TestStore_SaveSubscription(t *testing.T) {
	type args struct {
		sub store.Subscription
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid1",
			args: args{
				sub: store.Subscription{
					ID:             "id1",
					URL:            "url",
					CorrelationID:  "correlationID",
					ContentType:    "contentType",
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					UserID:         "userID",
					Type:           oapiConStore.Type_Resource,
				},
			},
		},
		{
			name: "valid2",
			args: args{
				sub: store.Subscription{
					ID:             "id2",
					URL:            "url",
					CorrelationID:  "correlationID",
					ContentType:    "contentType",
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					UserID:         "userID",
					Type:           oapiConStore.Type_Resource,
				},
			},
		},
		{
			name: "valid3",
			args: args{
				sub: store.Subscription{
					ID:             "id3",
					URL:            "url",
					CorrelationID:  "correlationID",
					ContentType:    "contentType",
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					UserID:         "userID",
					SigningSecret:  "signingSecret",
					Type:           oapiConStore.Type_Resource,
				},
			},
		},
		{
			name: "err",
			args: args{
				sub: store.Subscription{
					ID:             "id3",
					URL:            "url",
					CorrelationID:  "correlationID",
					ContentType:    "contentType",
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					UserID:         "userID",
					Type:           oapiConStore.Type_Resource,
				},
			},
			wantErr: true,
		},
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	ctx := context.Background()
	s := newStore(ctx, t, cfg)
	defer s.Close(ctx)
	defer s.Clear(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.SaveSubscription(context.Background(), tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

		})
	}
}

func TestStore_IncrementSubscriptionSequenceNumber(t *testing.T) {
	sub := store.Subscription{
		ID:             "id",
		URL:            "url",
		CorrelationID:  "correlationID",
		ContentType:    "contentType",
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		UserID:         "userID",
		Type:           oapiConStore.Type_Resource,
	}

	type args struct {
		subscriptionID string
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "valid1",
			args: args{
				subscriptionID: sub.ID,
			},
			want: 0,
		},
		{
			name: "valid2",
			args: args{
				subscriptionID: sub.ID,
			},
			want: 1,
		},
		{
			name: "err",
			args: args{
				subscriptionID: "id1",
			},
			wantErr: true,
		},
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	ctx := context.Background()
	s := newStore(ctx, t, cfg)
	defer s.Close(ctx)
	defer s.Clear(ctx)

	err = s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.IncrementSubscriptionSequenceNumber(ctx, tt.args.subscriptionID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestStore_PopSubscription(t *testing.T) {
	sub := store.Subscription{
		ID:             "id1",
		URL:            "url",
		CorrelationID:  "correlationID",
		ContentType:    "contentType",
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		UserID:         "userID",
		Type:           oapiConStore.Type_Resource,
	}

	type args struct {
		subscriptionID string
	}
	tests := []struct {
		name    string
		args    args
		wantSub store.Subscription
		wantErr bool
	}{
		{
			name: "valid1",
			args: args{
				subscriptionID: sub.ID,
			},
			wantSub: sub,
		},
		{
			name: "valid1",
			args: args{
				subscriptionID: sub.ID,
			},
			wantErr: true,
		},
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	ctx := context.Background()
	s := newStore(ctx, t, cfg)
	defer s.Close(ctx)
	defer s.Clear(ctx)

	err = s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSub, err := s.PopSubscription(ctx, tt.args.subscriptionID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantSub, gotSub)
			}
		})
	}
}

type testResourceHandler struct {
	data []store.Subscription
}

func (h *testResourceHandler) Handle(ctx context.Context, iter store.SubscriptionIter) (err error) {
	var sub store.Subscription
	for iter.Next(ctx, &sub) {
		h.data = append(h.data, sub)
	}
	return iter.Err()
}

func TestStore_LoadSubscriptions(t *testing.T) {
	sub := store.Subscription{
		ID:             "id",
		URL:            "url",
		CorrelationID:  "correlationID",
		ContentType:    "contentType",
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		UserID:         "userID",
		Type:           oapiConStore.Type_Resource,
	}

	type args struct {
		query store.SubscriptionQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []store.Subscription
	}{
		{
			name: "valid1",
			args: args{
				query: store.SubscriptionQuery{
					SubscriptionID: sub.ID,
				},
			},
			want: []store.Subscription{sub},
		},
		{
			name: "valid1",
			args: args{
				query: store.SubscriptionQuery{
					Type:     oapiConStore.Type_Resource,
					DeviceID: sub.DeviceID,
				},
			},
			want: []store.Subscription{sub},
		},
		{
			name: "valid2",
			args: args{
				query: store.SubscriptionQuery{
					Type:     oapiConStore.Type_Resource,
					DeviceID: sub.DeviceID,
					Href:     sub.Href,
				},
			},
			want: []store.Subscription{sub},
		},
		{
			name: "empty",
			args: args{
				query: store.SubscriptionQuery{
					SubscriptionID: "empty",
				},
			},
		},
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	ctx := context.Background()
	s := newStore(ctx, t, cfg)
	defer s.Close(ctx)
	defer s.Clear(ctx)

	err = s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testResourceHandler
			err = s.LoadSubscriptions(ctx, tt.args.query, &h)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, h.data)
			}
		})
	}
}
