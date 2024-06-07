package mongodb_test

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreGetConfigurations(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	confs := test.AddConfigurationsToStore(ctx, t, s, 500, nil)

	type args struct {
		owner string
		query *pb.GetConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, configurations []*store.Configuration)
	}{
		{
			name: "all",
			args: args{},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, len(confs))
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					test.CmpStoredConfiguration(t, &conf, c, true, true)
				}
			},
		},
		{
			name: "owner0",
			args: args{
				owner: test.Owner(0),
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.NotEmpty(t, configurations)
				for _, c := range configurations {
					require.Equal(t, test.Owner(0), c.Owner)
					conf, ok := confs[c.Id]
					require.True(t, ok)
					test.CmpStoredConfiguration(t, &conf, c, true, true)
				}
			},
		},
		{
			name: "id1/all",
			args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(1),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 1)
				c := configurations[0]
				conf, ok := confs[c.Id]
				require.True(t, ok)
				test.CmpStoredConfiguration(t, &conf, c, true, true)
			},
		},
		{
			name: "owner2/id2/all",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(2),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 1)
				c := configurations[0]
				conf, ok := confs[c.Id]
				require.True(t, ok)
				require.Equal(t, test.ConfigurationID(2), conf.Id)
				require.Equal(t, test.Owner(2), conf.Owner)
				test.CmpStoredConfiguration(t, &conf, c, true, true)
			},
		},
		{
			name: "latest",
			args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, test.RuntimeConfig.NumConfigurations)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, conf.Id, c.Id)
					require.Equal(t, conf.Owner, c.Owner)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, conf.Latest, c.Latest)
				}
			},
		},
		{
			name: "owner1/latest",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(1), conf.Owner)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, conf.Latest, c.Latest)
				}
			},
		},
		{
			name: "owner2/id1/latest - non-matching owner",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetConfigurationsRequest{
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
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Empty(t, configurations)
			},
		},
		{
			name: "owner2{latest, id2/latest, id5/latest}", args: args{
				owner: test.Owner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: test.ConfigurationID(2),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: test.ConfigurationID(5),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(2), conf.Owner)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, conf.Latest, c.Latest)
				}
			},
		},
		{
			name: "version/42",
			args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 42,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 10)
				for _, c := range configurations {
					_, ok := confs[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 1)
					require.Equal(t, uint64(42), c.Versions[0].Version)
				}
			},
		},
		{
			name: "owner2/version/{13, 37, 42}", args: args{
				owner: test.Owner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 13,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 42,
							},
						},
						// duplicates should be ignored
						{
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
						// filter with Id should be ignored if there are filters without Id
						{
							Id: test.ConfigurationID(2),
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					_, ok := confs[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 3)
					slices.SortFunc(c.Versions, func(i, j store.ConfigurationVersion) int {
						return cmp.Compare(i.Version, j.Version)
					})
					require.Equal(t, uint64(13), c.Versions[0].Version)
					require.Equal(t, uint64(37), c.Versions[1].Version)
					require.Equal(t, uint64(42), c.Versions[2].Version)
				}
			},
		},
		{
			name: "id0/version/{1..max} + latest",
			args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: func() []*pb.IDFilter {
						var idFilters []*pb.IDFilter
						c := confs[test.ConfigurationID(0)]
						for _, v := range c.Versions {
							idFilters = append(idFilters, &pb.IDFilter{
								Id: test.ConfigurationID(0),
								Version: &pb.IDFilter_Value{
									Value: v.Version,
								},
							})
						}
						idFilters = append(idFilters, &pb.IDFilter{
							Id: test.ConfigurationID(0),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						})
						return idFilters
					}(),
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 1)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, test.ConfigurationID(0), conf.Id)
					test.CmpStoredConfiguration(t, &conf, c, true, false)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configurations []*store.Configuration
			err := s.GetConfigurations(ctx, tt.args.owner, tt.args.query, func(iterCtx context.Context, iter store.Iterator[store.Configuration]) error {
				var conf store.Configuration
				for iter.Next(iterCtx, &conf) {
					configurations = append(configurations, conf.Clone())
				}
				return iter.Err()
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, configurations)
		})
	}
}
