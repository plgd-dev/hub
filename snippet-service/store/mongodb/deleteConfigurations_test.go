package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreDeleteConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	getConfigurations := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetConfigurationsRequest) []*store.Configuration {
		var configurations []*store.Configuration
		err := s.GetConfigurations(ctx, owner, query, func(iterCtx context.Context, iter store.Iterator[store.Configuration]) error {
			var conf store.Configuration
			for iter.Next(iterCtx, &conf) {
				configurations = append(configurations, conf.Clone())
			}
			return iter.Err()
		})
		require.NoError(t, err)
		return configurations
	}

	type args struct {
		owner string
		query *pb.DeleteConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration)
	}{
		{
			name: "all",
			args: args{},
			want: func(t *testing.T, s *mongodb.Store, _ map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				require.Empty(t, confs)
			},
		},
		{
			name: "owner1",
			args: args{
				owner: test.Owner(1),
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				require.NotEmpty(t, confs)
				newCount := 0
				for _, conf := range confs {
					require.NotEqual(t, test.Owner(1), conf.Owner)
					newCount += len(conf.Versions)
				}
				storedCount := 0
				for _, conf := range stored {
					if conf.Owner != test.Owner(1) {
						storedCount += len(conf.Versions)
					}
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "id{1,3,4,5}",
			args: args{
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id:      test.ConfigurationID(1),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConfigurationID(3),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConfigurationID(4),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConfigurationID(5),
							Version: &pb.IDFilter_All{All: true},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				require.NotEmpty(t, confs)
				newCount := 0
				for _, conf := range confs {
					require.NotEqual(t, test.ConfigurationID(1), conf.Id)
					require.NotEqual(t, test.ConfigurationID(3), conf.Id)
					require.NotEqual(t, test.ConfigurationID(4), conf.Id)
					require.NotEqual(t, test.ConfigurationID(5), conf.Id)
					newCount += len(conf.Versions)
				}
				storedCount := 0
				for _, conf := range stored {
					if conf.Id == test.ConfigurationID(1) ||
						conf.Id == test.ConfigurationID(3) ||
						conf.Id == test.ConfigurationID(4) ||
						conf.Id == test.ConfigurationID(5) {
						continue
					}
					storedCount += len(conf.Versions)
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "owner2/id2",
			args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id:      test.ConfigurationID(2),
							Version: &pb.IDFilter_All{All: true},
						},
						// Ids not owned by owner2 should not be deleted
						{
							Id:      test.ConfigurationID(1),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConfigurationID(3),
							Version: &pb.IDFilter_All{All: true},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				require.NotEmpty(t, confs)
				newCount := 0
				for _, conf := range confs {
					require.NotEqual(t, test.ConfigurationID(2), conf.Id)
					newCount += len(conf.Versions)
				}
				storedCount := 0
				for _, conf := range stored {
					if conf.Id == test.ConfigurationID(2) {
						continue
					}
					storedCount += len(conf.Versions)
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "latest",
			args: args{
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				storedLatest := make(map[string]store.ConfigurationVersion)
				storedCount := 0
				for _, conf := range stored {
					storedLatest[conf.Id] = conf.Versions[len(conf.Versions)-1]
					storedCount += len(conf.Versions)
				}
				confs := getConfigurations(t, s, "", nil)
				require.NotEmpty(t, confs)
				count := 0
				for _, conf := range confs {
					require.NotEqual(t, storedLatest[conf.Id], conf.Versions[len(conf.Versions)-1])
					count += len(conf.Versions)
				}
				require.Equal(t, storedCount-len(storedLatest), count)
			},
		},
		{
			name: "owner1/latest",
			args: args{
				owner: test.Owner(1),
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						// duplicates should be ignored
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				storedLatest := make(map[string]store.ConfigurationVersion)
				storedCount := 0
				for _, conf := range stored {
					storedLatest[conf.Id] = conf.Versions[len(conf.Versions)-1]
					storedCount += len(conf.Versions)
				}
				confs := getConfigurations(t, s, "", nil)
				require.NotEmpty(t, confs)
				count := 0
				removed := 0
				for _, conf := range confs {
					if conf.Owner == test.Owner(1) {
						require.NotEqual(t, storedLatest[conf.Id], conf.Versions[len(conf.Versions)-1])
						removed++
					} else {
						require.Equal(t, storedLatest[conf.Id], conf.Versions[len(conf.Versions)-1])
					}
					count += len(conf.Versions)
				}
				require.Equal(t, storedCount-removed, count)
			},
		},
		{
			name: "owner2/id1/latest - non-matching owner",
			args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(1),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				confsMap := make(map[string]store.Configuration)
				for _, conf := range confs {
					confsMap[conf.Id] = *conf
				}
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
			},
		},
		{
			name: "version/{42, 142, 242, 342, 442, 542}", args: args{
				owner: "",
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{Version: &pb.IDFilter_Value{Value: 42}},
						{Version: &pb.IDFilter_Value{Value: 142}},
						{Version: &pb.IDFilter_Value{Value: 242}},
						{Version: &pb.IDFilter_Value{Value: 342}},
						{Version: &pb.IDFilter_Value{Value: 442}},
						{Version: &pb.IDFilter_Value{Value: 542}},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				confsMap := make(map[string]store.Configuration)
				for _, conf := range confs {
					confsMap[conf.Id] = *conf
				}
				for _, conf := range stored {
					versions := make([]store.ConfigurationVersion, 0)
					for _, version := range conf.Versions {
						if version.Version == 42 ||
							version.Version == 142 ||
							version.Version == 242 ||
							version.Version == 342 ||
							version.Version == 442 ||
							version.Version == 542 {
							continue
						}
						versions = append(versions, version)
					}
					conf.Versions = versions
					stored[conf.Id] = conf
				}
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
			},
		},
		{
			name: "owner2/version/{213, 237, 242}", args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{Version: &pb.IDFilter_Value{Value: 213}},
						{Version: &pb.IDFilter_Value{Value: 237}},
						{Version: &pb.IDFilter_Value{Value: 242}},
						// duplicates should be ignored
						{Version: &pb.IDFilter_Value{Value: 237}},
						// filter with Id should be ignored if there are filters without Id
						{
							Id:      test.ConfigurationID(2),
							Version: &pb.IDFilter_Value{Value: 237},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", nil)
				confsMap := make(map[string]store.Configuration)
				for _, conf := range confs {
					confsMap[conf.Id] = *conf
				}
				for _, conf := range stored {
					if conf.Owner == test.Owner(2) {
						versions := make([]store.ConfigurationVersion, 0)
						for _, version := range conf.Versions {
							if version.Version == 213 ||
								version.Version == 237 ||
								version.Version == 242 {
								continue
							}
							versions = append(versions, version)
						}
						conf.Versions = versions
						stored[conf.Id] = conf
					}
				}
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, cleanUpStore := test.NewMongoStore(t)
			defer cleanUpStore()
			inserted := test.AddConfigurationsToStore(ctx, t, s, 500, func(iteration int) uint64 {
				return uint64(iteration * 100)
			})
			_, err := s.DeleteConfigurations(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, s, inserted)
		})
	}
}
