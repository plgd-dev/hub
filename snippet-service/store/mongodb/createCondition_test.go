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

func TestStoreCreateCondition(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	condID := uuid.NewString()
	confID := uuid.NewString()

	type args struct {
		cond *pb.Condition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(*testing.T, *pb.Condition)
	}{
		{
			name: "invalid ID",
			args: args{
				cond: &pb.Condition{
					Id:              "invalid",
					ConfigurationId: uuid.NewString(),
					Owner:           "owner",
				},
			},
			wantErr: true,
		},
		{
			name: "missing ConfigurationId",
			args: args{
				cond: &pb.Condition{
					Id:    uuid.NewString(),
					Owner: "owner",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ConfigurationId",
			args: args{
				cond: &pb.Condition{
					Id:              uuid.NewString(),
					ConfigurationId: "invalid",
					Owner:           "owner",
				},
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: args{
				cond: &pb.Condition{
					Id:              uuid.NewString(),
					ConfigurationId: uuid.NewString(),
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				cond: &pb.Condition{
					Id:              condID,
					ConfigurationId: confID,
					Owner:           "owner",
				},
			},
			want: func(t *testing.T, got *pb.Condition) {
				want := &pb.Condition{
					Id:              condID,
					ConfigurationId: confID,
					Owner:           "owner",
				}
				test.CmpCondition(t, want, got, true)
			},
		},
		{
			name: "valid (generated ID)",
			args: args{
				cond: &pb.Condition{
					ConfigurationId: uuid.NewString(),
					Owner:           "owner",
				},
			},
			want: func(t *testing.T, got *pb.Condition) {
				want := &pb.Condition{
					Id:              got.GetId(),
					ConfigurationId: got.GetConfigurationId(),
					Owner:           "owner",
				}
				test.CmpCondition(t, want, got, true)
			},
		},
		{
			name: "duplicit ID",
			args: args{
				cond: &pb.Condition{
					Id:              condID,
					ConfigurationId: uuid.NewString(),
					Owner:           "owner",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.CreateCondition(ctx, tt.args.cond)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, got)
		})
	}
}
