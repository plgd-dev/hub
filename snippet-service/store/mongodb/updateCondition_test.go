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

	cond, err := s.CreateCondition(ctx, &pb.Condition{
		Id:              uuid.NewString(),
		Name:            "update",
		Enabled:         true,
		ConfigurationId: uuid.NewString(),
		Owner:           "owner1",
		Version:         42,
	})
	require.NoError(t, err)

	type args struct {
		update *pb.Condition
	}
	tests := []struct {
		name    string
		args    args
		want    func(*pb.Condition)
		wantErr bool
	}{
		{
			name: "missing Id",
			args: args{
				update: &pb.Condition{
					Owner:           cond.GetOwner(),
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
					Owner:   cond.GetOwner(),
					Version: 1,
				},
			},
			wantErr: true,
		},
		{
			name: "non-matching owner",
			args: args{
				update: &pb.Condition{
					Id:              cond.GetId(),
					Owner:           "invalid",
					ConfigurationId: cond.GetConfigurationId(),
					Version:         1,
				},
			},
			wantErr: true,
		},
		{
			name: "duplicit version",
			args: args{
				update: &pb.Condition{
					Id:              cond.GetId(),
					Owner:           cond.GetOwner(),
					ConfigurationId: cond.GetConfigurationId(),
					Version:         cond.GetVersion(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid version",
			args: args{
				update: &pb.Condition{
					Id:              cond.GetId(),
					Owner:           cond.GetOwner(),
					ConfigurationId: cond.GetConfigurationId(),
					Version:         cond.GetVersion() - 1, // version must be higher than the latest one
				},
			},
			wantErr: true,
		},
		{
			name: "update",
			args: args{
				update: &pb.Condition{
					Id:                 cond.GetId(),
					ConfigurationId:    cond.GetConfigurationId(),
					Owner:              cond.GetOwner(),
					Version:            43,
					Name:               "updated name",
					Enabled:            false,
					ApiAccessToken:     "updated token",
					DeviceIdFilter:     []string{"device2", "device1", "device1"},
					ResourceTypeFilter: []string{"plgd.test", "plgd.test"},
					ResourceHrefFilter: []string{"/test/2", "/test/1", "/test/2"},
					JqExpressionFilter: "{}",
				},
			},
			want: func(updated *pb.Condition) {
				require.Equal(t, "updated name", updated.GetName())
				require.False(t, updated.GetEnabled())
				require.Equal(t, "updated token", updated.GetApiAccessToken())
				require.ElementsMatch(t, []string{"device1", "device2"}, updated.GetDeviceIdFilter())
				require.ElementsMatch(t, []string{"plgd.test"}, updated.GetResourceTypeFilter())
				require.ElementsMatch(t, []string{"/test/1", "/test/2"}, updated.GetResourceHrefFilter())
				require.Equal(t, "{}", updated.GetJqExpressionFilter())
			},
		},
		{
			name: "second update",
			args: args{
				update: &pb.Condition{
					Id:              cond.GetId(),
					ConfigurationId: cond.GetConfigurationId(),
					Owner:           cond.GetOwner(),
					Name:            "next updated name",
					Enabled:         true,
					ApiAccessToken:  "next updated token",
				},
			},
			want: func(updated *pb.Condition) {
				require.Equal(t, "next updated name", updated.GetName())
				require.True(t, updated.GetEnabled())
				require.Equal(t, "next updated token", updated.GetApiAccessToken())
				require.Empty(t, updated.GetDeviceIdFilter())
				require.Empty(t, updated.GetResourceTypeFilter())
				require.Empty(t, updated.GetResourceHrefFilter())
				require.Empty(t, updated.GetJqExpressionFilter())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, err := s.UpdateCondition(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(cond)
		})
	}
}
