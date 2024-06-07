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

func TestStoreDeleteConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

	getConfigurationsMap := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetConfigurationsRequest) map[string]store.Configuration {
		confs := getConfigurations(t, s, owner, query)
		confsMap := make(map[string]store.Configuration)
		for _, conf := range confs {
			confsMap[conf.Id] = *conf
		}
		return confsMap
	}

	type args struct {
		owner     string
		query     *pb.DeleteConfigurationsRequest
		makeQuery func(stored map[string]store.Configuration) *pb.DeleteConfigurationsRequest
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
				confsMap := getConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]store.Configuration)
				for _, conf := range stored {
					if conf.Owner == test.Owner(1) {
						continue
					}
					newStored[conf.Id] = conf
				}
				test.CmpStoredConfigurationMaps(t, newStored, confsMap)
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
				confsMap := getConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]store.Configuration)
				for _, conf := range stored {
					if conf.Id == test.ConfigurationID(1) ||
						conf.Id == test.ConfigurationID(3) ||
						conf.Id == test.ConfigurationID(4) ||
						conf.Id == test.ConfigurationID(5) {
						continue
					}
					newStored[conf.Id] = conf
				}
				test.CmpStoredConfigurationMaps(t, newStored, confsMap)
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
				confsMap := getConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				newStored := make(map[string]store.Configuration)
				for _, conf := range stored {
					if conf.Id == test.ConfigurationID(2) {
						continue
					}
					newStored[conf.Id] = conf
				}
				test.CmpStoredConfigurationMaps(t, newStored, confsMap)
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
				for id, conf := range stored {
					conf.Versions = conf.Versions[:len(conf.Versions)-1]
					latest := conf.Versions[len(conf.Versions)-1].Copy()
					conf.Latest = &latest
					stored[id] = conf
				}
				confsMap := getConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
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
				for id, conf := range stored {
					if conf.Owner != test.Owner(1) {
						continue
					}
					conf.Versions = conf.Versions[:len(conf.Versions)-1]
					latest := conf.Versions[len(conf.Versions)-1].Copy()
					conf.Latest = &latest
					stored[id] = conf
				}
				confsMap := getConfigurationsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
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
				confsMap := getConfigurationsMap(t, s, "", nil)
				test.CmpStoredConfigurationMaps(t, stored, confsMap)
			},
		},
		{
			name: "version/{42, 142, 242, 342, 442, 542}",
			args: args{
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
				confsMap := getConfigurationsMap(t, s, "", nil)
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
			name: "owner2/version/{213, 237, 242}",
			args: args{
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
				confsMap := getConfigurationsMap(t, s, "", nil)
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
		{
			name: "id7/version/{$(all)}",
			args: args{
				makeQuery: func(stored map[string]store.Configuration) *pb.DeleteConfigurationsRequest {
					r := &pb.DeleteConfigurationsRequest{}
					id7, ok := stored[test.ConfigurationID(7)]
					require.True(t, ok)
					for _, version := range id7.Versions {
						r.IdFilter = append(r.IdFilter, &pb.IDFilter{
							Id:      test.ConfigurationID(7),
							Version: &pb.IDFilter_Value{Value: version.Version},
						})
					}
					return r
				},
			},
			want: func(t *testing.T, s *mongodb.Store, _ map[string]store.Configuration) {
				confs := getConfigurations(t, s, "", &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(7),
						},
					},
				})
				require.Empty(t, confs)
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
			query := tt.args.query
			if tt.args.makeQuery != nil {
				query = tt.args.makeQuery(inserted)
			}
			_, err := s.DeleteConfigurations(ctx, tt.args.owner, query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, s, inserted)
		})
	}
}
