package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store/mongodb"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func newTestStore(t *testing.T) (*mongodb.Store, func()) {
	cfg := test.MakeConfig(t)

	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	certManager, err := client.New(cfg.Clients.Storage.MongoDB.TLS, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	ctx := context.Background()
	s, err := mongodb.NewStore(ctx, cfg.Clients.Storage.MongoDB, certManager.GetTLSConfig(), noop.NewTracerProvider())
	require.NoError(t, err)

	return s, func() {
		err := s.Clear(ctx)
		require.NoError(t, err)
		_ = s.Close(ctx)
		certManager.Close()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

func TestStoreSaveSubscription(t *testing.T) {
	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()
	token := oauthTest.GetDefaultAccessToken(t)

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
					Accept:         []string{"accept"},
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					Type:           store.Type_Resource,
					AccessToken:    token,
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
					Accept:         []string{"accept"},
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					Type:           store.Type_Resource,
					AccessToken:    token,
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
					Accept:         []string{"accept"},
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					Type:           store.Type_Resource,
					AccessToken:    token,
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
					Accept:         []string{"accept"},
					EventTypes:     []events.EventType{"eventtypes"},
					SequenceNumber: 0,
					DeviceID:       "deviceid",
					Href:           "resourcehref",
					SigningSecret:  "signingSecret",
					Type:           store.Type_Resource,
					AccessToken:    token,
				},
			},
			wantErr: true,
		},
	}

	ctx := context.Background()
	s, cleanUp := newTestStore(t)
	defer cleanUp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.SaveSubscription(ctx, tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreIncrementSubscriptionSequenceNumber(t *testing.T) {
	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()
	token := oauthTest.GetDefaultAccessToken(t)

	sub := store.Subscription{
		ID:             "id",
		URL:            "url",
		CorrelationID:  "correlationID",
		Accept:         []string{"accept"},
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		Type:           store.Type_Resource,
		AccessToken:    token,
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

	ctx := context.Background()
	s, cleanUp := newTestStore(t)
	defer cleanUp()

	err := s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.IncrementSubscriptionSequenceNumber(ctx, tt.args.subscriptionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStorePopSubscription(t *testing.T) {
	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()
	token := oauthTest.GetDefaultAccessToken(t)

	sub := store.Subscription{
		ID:             "id1",
		URL:            "url",
		CorrelationID:  "correlationID",
		Accept:         []string{"accept"},
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		Type:           store.Type_Resource,
		AccessToken:    token,
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

	ctx := context.Background()
	s, cleanUp := newTestStore(t)
	defer cleanUp()

	err := s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSub, err := s.PopSubscription(ctx, tt.args.subscriptionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSub, gotSub)
		})
	}
}

type testResourceHandler struct {
	data []store.Subscription
}

func (h *testResourceHandler) Handle(ctx context.Context, iter store.SubscriptionIter) (err error) {
	for {
		var sub store.Subscription
		if !iter.Next(ctx, &sub) {
			break
		}
		h.data = append(h.data, sub)
	}
	return iter.Err()
}

func TestStoreLoadSubscriptions(t *testing.T) {
	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()
	token := oauthTest.GetDefaultAccessToken(t)

	sub := store.Subscription{
		ID:             "id",
		URL:            "url",
		CorrelationID:  "correlationID",
		Accept:         []string{"accept"},
		EventTypes:     []events.EventType{"eventtypes"},
		SequenceNumber: 0,
		DeviceID:       "deviceid",
		Href:           "resourcehref",
		SigningSecret:  "signingSecret",
		Type:           store.Type_Resource,
		AccessToken:    token,
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
					Type:     store.Type_Resource,
					DeviceID: sub.DeviceID,
				},
			},
			want: []store.Subscription{sub},
		},
		{
			name: "valid2",
			args: args{
				query: store.SubscriptionQuery{
					Type:     store.Type_Resource,
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

	ctx := context.Background()
	s, cleanUp := newTestStore(t)
	defer cleanUp()

	err := s.SaveSubscription(context.Background(), sub)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testResourceHandler
			err = s.LoadSubscriptions(ctx, tt.args.query, &h)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, h.data)
		})
	}
}
