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

	confID := uuid.NewString()
	const owner = "owner1"
	resources := []*pb.Configuration_Resource{
		makeLightResourceConfiguration(t, "1", 1, 1337),
	}
	conf, err := s.CreateConfiguration(ctx, &pb.Configuration{
		Id:        confID,
		Name:      "valid",
		Owner:     owner,
		Version:   13,
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
					Id:        conf.GetId(),
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
					Id:        conf.GetId(),
					Owner:     conf.GetOwner(),
					Version:   13,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid version",
			args: args{
				update: &pb.Configuration{
					Id:        conf.GetId(),
					Owner:     conf.GetOwner(),
					Version:   conf.GetVersion() - 1, // version must be higher than the latest one
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing ID",
			args: args{
				update: &pb.Configuration{
					Owner:     conf.GetOwner(),
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
					Id:    conf.GetId(),
					Name:  "updated name",
					Owner: conf.GetOwner(),
					Resources: []*pb.Configuration_Resource{
						makeLightResourceConfiguration(t, "2", 2, 42),
					},
				},
			},
			want: func(t *testing.T, updatedConf *pb.Configuration) {
				require.Equal(t, conf.GetId(), updatedConf.GetId())
				require.Equal(t, "updated name", updatedConf.GetName())
				require.Equal(t, conf.GetOwner(), updatedConf.GetOwner())
				require.Equal(t, conf.GetVersion()+1, updatedConf.GetVersion())
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
