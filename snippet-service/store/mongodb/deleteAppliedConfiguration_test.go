package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/stretchr/testify/require"
)

func TestStoreDeleteAppliedConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	getAppliedConfigurations := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetAppliedConfigurationsRequest) []*pb.AppliedConfiguration {
		var configurations []*pb.AppliedConfiguration
		err := s.GetAppliedConfigurations(ctx, owner, query, func(c *store.AppliedConfiguration) error {
			configurations = append(configurations, c.GetAppliedConfiguration().Clone())
			return nil
		})
		require.NoError(t, err)
		return configurations
	}

	getAppliedConfigurationsMap := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetAppliedConfigurationsRequest) map[string]*pb.AppliedConfiguration {
		confs := getAppliedConfigurations(t, s, owner, query)
		confsMap := make(map[string]*pb.AppliedConfiguration)
		for _, conf := range confs {
			confsMap[conf.GetId()] = conf
		}
		return confsMap
	}

	type args struct {
		owner string
		query *pb.DeleteAppliedConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, s *mongodb.Store, stored map[string]*pb.AppliedConfiguration)
	}{
		{
			name: "all",
			args: args{},
			want: func(t *testing.T, s *mongodb.Store, _ map[string]*pb.AppliedConfiguration) {
				confs := getAppliedConfigurations(t, s, "", nil)
				require.Empty(t, confs)
			},
		},
		{
			name: "owner2",
			args: args{
				owner: test.Owner(1),
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]*pb.AppliedConfiguration) {
				confsMap := getAppliedConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]*pb.AppliedConfiguration)
				for _, conf := range stored {
					if conf.GetOwner() == test.Owner(1) {
						continue
					}
					newStored[conf.GetId()] = conf
				}
				test.CmpAppliedDeviceConfigurationsMaps(t, newStored, confsMap, false)
			},
		},
		{
			name: "id{1,3,4,5}",
			args: args{
				query: &pb.DeleteAppliedConfigurationsRequest{
					IdFilter: []string{
						test.AppliedConfigurationID(1),
						test.AppliedConfigurationID(3),
						test.AppliedConfigurationID(4),
						test.AppliedConfigurationID(5),
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]*pb.AppliedConfiguration) {
				confsMap := getAppliedConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]*pb.AppliedConfiguration)

				for _, conf := range stored {
					confID := conf.GetId()
					if confID == test.AppliedConfigurationID(1) ||
						confID == test.AppliedConfigurationID(3) ||
						confID == test.AppliedConfigurationID(4) ||
						confID == test.AppliedConfigurationID(5) {
						continue
					}
					newStored[confID] = conf
				}
				test.CmpAppliedDeviceConfigurationsMaps(t, newStored, confsMap, false)
			},
		},
		{
			name: "owner2/id2",
			args: args{
				owner: test.Owner(2),
				query: &pb.DeleteAppliedConfigurationsRequest{
					IdFilter: []string{
						test.AppliedConfigurationID(2),
						// Ids not owned by owner2 should not be deleted
						test.AppliedConfigurationID(1),
						test.AppliedConfigurationID(3),
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]*pb.AppliedConfiguration) {
				confsMap := getAppliedConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]*pb.AppliedConfiguration)
				for _, conf := range stored {
					confID := conf.GetId()
					if confID == test.AppliedConfigurationID(2) {
						continue
					}
					newStored[confID] = conf
				}
				test.CmpAppliedDeviceConfigurationsMaps(t, newStored, confsMap, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, cleanUpStore := test.NewMongoStore(t)
			defer cleanUpStore()
			inserted := test.AddAppliedConfigurationsToStore(ctx, t, s)
			err := s.DeleteAppliedConfigurations(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			stored := make(map[string]*pb.AppliedConfiguration)
			for id, conf := range inserted {
				stored[id] = conf.GetAppliedConfiguration()
			}
			tt.want(t, s, stored)
		})
	}
}
