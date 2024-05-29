package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreUpdateCondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	condID := uuid.NewString()
	confID := uuid.NewString()
	const owner = "owner1"

	create := func() {
		_, err := s.CreateCondition(ctx, &pb.Condition{
			Id:              condID,
			Name:            "update",
			Enabled:         true,
			ConfigurationId: confID,
			Owner:           owner,
		})
		require.NoError(t, err)
	}
	create()

	type args struct {
		update *pb.Condition
		reset  bool
	}
	tests := []struct {
		name    string
		args    args
		want    func(*testing.T, *pb.Condition)
		wantErr bool
	}{
		{
			name: "missing Id",
			args: args{
				update: &pb.Condition{
					Owner:           owner,
					ConfigurationId: uuid.NewString(),
					Version:         1,
				},
			},
			wantErr: true,
		},
		{
			name: "missing ConfigurationId",
			args: args{
				update: &pb.Condition{
					Id:      uuid.NewString(),
					Owner:   owner,
					Version: 1,
				},
			},
			wantErr: true,
		},
		{
			name: "duplicit version",
			args: args{
				update: &pb.Condition{
					Id:              condID,
					Owner:           owner,
					ConfigurationId: uuid.NewString(),
					Version:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "non-matching owner",
			args: args{
				update: &pb.Condition{
					Id:              condID,
					Owner:           "invalid",
					ConfigurationId: confID,
					Version:         1,
				},
			},
			wantErr: true,
		},
		{
			name: "update",
			args: args{
				update: &pb.Condition{
					Id:                 condID,
					ConfigurationId:    confID,
					Owner:              owner,
					Version:            1,
					Name:               "updated name",
					Enabled:            false,
					ApiAccessToken:     "updated token",
					DeviceIdFilter:     []string{"device2", "device1", "device1"},
					ResourceTypeFilter: []string{"plgd.test", "plgd.test"},
					ResourceHrefFilter: []string{"/test/2", "/test/1", "/test/2"},
					JqExpressionFilter: "{}",
				},
			},
			want: func(*testing.T, *pb.Condition) {
				// TODO: check that the values are updated in the DB
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.reset {
				err := s.Clear(ctx)
				require.NoError(t, err)
				create()
			}

			cond, err := s.UpdateCondition(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, cond)
		})
	}
}
