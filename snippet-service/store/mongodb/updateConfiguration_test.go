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

func TestStoreUpdateConfiguration(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	confID := uuid.New().String()
	const owner = "owner1"
	resources := []*pb.Configuration_Resource{
		makeLightResourceConfiguration(t, "1", 1, 1337),
	}
	_, err := s.CreateConfiguration(ctx, &pb.Configuration{
		Id:        confID,
		Name:      "valid",
		Owner:     owner,
		Version:   0,
		Resources: resources,
	})
	require.NoError(t, err)

	type args struct {
		update *pb.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(*testing.T, *pb.Configuration)
	}{
		{
			name: "non-matching owner",
			args: args{
				update: &pb.Configuration{
					Id:        confID,
					Owner:     "invalid",
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "duplicit version",
			args: args{
				update: &pb.Configuration{
					Id:        confID,
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing ID",
			args: args{
				update: &pb.Configuration{
					Owner:     owner,
					Version:   1,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				update: &pb.Configuration{
					Id:      confID,
					Owner:   owner,
					Version: 1,
					Resources: []*pb.Configuration_Resource{
						makeLightResourceConfiguration(t, "2", 2, 42),
					},
				},
			},
			want: func(*testing.T, *pb.Configuration) {
				// TODO: check the updated configuration
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := s.UpdateConfiguration(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, conf)
		})
	}
}
