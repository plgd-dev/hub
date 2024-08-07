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

func TestStoreGetConditions(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	conds := test.AddConditionsToStore(ctx, t, s, 500, nil)

	type args struct {
		owner string
		query *pb.GetConditionsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, conditions map[string]store.Condition)
	}{
		{
			name: "all",
			args: args{},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, len(conds))
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					test.CmpStoredCondition(t, &cond, &c, false, true)
				}
			},
		},
		{
			name: "owner0",
			args: args{
				owner: test.Owner(0),
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.NotEmpty(t, conditions)
				for _, c := range conditions {
					require.Equal(t, test.Owner(0), c.Owner)
					cond, ok := conds[c.Id]
					require.True(t, ok)
					test.CmpStoredCondition(t, &cond, &c, false, true)
				}
			},
		},
		{
			name: "id1/all",
			args: args{
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(1),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 1)
				c, ok := conditions[test.ConditionID(1)]
				require.True(t, ok)
				cond, ok := conds[c.Id]
				require.True(t, ok)
				test.CmpStoredCondition(t, &cond, &c, false, true)
			},
		},
		{
			name: "owner2/id2/all",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(2),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 1)
				c, ok := conditions[test.ConditionID(2)]
				require.True(t, ok)
				cond, ok := conds[c.Id]
				require.True(t, ok)
				require.Equal(t, test.ConditionID(2), cond.Id)
				require.Equal(t, test.Owner(2), cond.Owner)
				test.CmpStoredCondition(t, &cond, &c, false, true)
			},
		},
		{
			name: "latest",
			args: args{
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, test.RuntimeConfig.NumConditions)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, cond.Id, c.Id)
					require.Equal(t, cond.Owner, c.Owner)
					require.Equal(t, cond.ConfigurationId, c.ConfigurationId)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, cond.Latest, c.Latest)
				}
			},
		},
		{
			name: "owner1/latest",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 7)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(1), cond.Owner)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, cond.Latest, c.Latest)
				}
			},
		},
		{
			name: "owner2/id1/latest - non-matching owner",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(1),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Empty(t, conditions)
			},
		},
		{
			name: "owner2{latest, id2/latest, id5/latest} - non-matching owner",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: test.ConditionID(2),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: test.ConditionID(5),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 6)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(2), cond.Owner)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, cond.Latest, c.Latest)
				}
			},
		},
		{
			name: "version/42", args: args{
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 13,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, test.RuntimeConfig.NumConditions)
				for _, c := range conditions {
					_, ok := conds[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 1)
					require.Equal(t, uint64(13), c.Versions[0].Version)
				}
			},
		},
		{
			name: "owner2/version/{7, 13, 19}", args: args{
				owner: test.Owner(2),
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 7,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 13,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 19,
							},
						},
						// duplicates should be ignored
						{
							Version: &pb.IDFilter_Value{
								Value: 7,
							},
						},
						// filter with Id should be ignored if there are filters without Id
						{
							Id: test.ConditionID(2),
							Version: &pb.IDFilter_Value{
								Value: 19,
							},
						},
					},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 6)
				for _, c := range conditions {
					_, ok := conds[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 3)
					slices.SortFunc(c.Versions, func(i, j store.ConditionVersion) int {
						return cmp.Compare(i.Version, j.Version)
					})
					require.Equal(t, uint64(7), c.Versions[0].Version)
					require.Equal(t, uint64(13), c.Versions[1].Version)
					require.Equal(t, uint64(19), c.Versions[2].Version)
				}
			},
		},
		{
			name: "id0/version/{1..max} + latest",
			args: args{
				query: &pb.GetConditionsRequest{
					IdFilter: func() []*pb.IDFilter {
						var idFilters []*pb.IDFilter
						c := conds[test.ConditionID(0)]
						for _, v := range c.Versions {
							idFilters = append(idFilters, &pb.IDFilter{
								Id: test.ConditionID(0),
								Version: &pb.IDFilter_Value{
									Value: v.Version,
								},
							})
						}
						idFilters = append(idFilters, &pb.IDFilter{
							Id: test.ConditionID(0),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						})
						return idFilters
					}(),
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.Len(t, conditions, 1)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.ConditionID(0), cond.Id)
					test.CmpStoredCondition(t, &cond, &c, true, false)
				}
			},
		},
		{
			name: "confId3/latest",
			args: args{
				query: &pb.GetConditionsRequest{
					ConfigurationIdFilter: []string{test.ConfigurationID(3)},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				require.NotEmpty(t, conditions)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.ConfigurationID(3), cond.ConfigurationId)
					require.Empty(t, c.Versions)
					test.CmpJSON(t, cond.Latest, c.Latest)
				}
			},
		},
		{
			name: "owner1/{confId1, confId3}/latest",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetConditionsRequest{
					ConfigurationIdFilter: []string{test.ConfigurationID(1), test.ConfigurationID(3)},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				stored := make(map[string]store.Condition)
				for _, c := range conds {
					if c.Owner == test.Owner(1) &&
						(c.ConfigurationId == test.ConfigurationID(1) || c.ConfigurationId == test.ConfigurationID(3)) {
						c.Versions = nil
						stored[c.Id] = c
					}
				}
				test.CmpStoredConditionMaps(t, stored, conditions)
			},
		},
		{
			name: "id0/latest + confId1/latest",
			args: args{
				query: &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(0),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
					ConfigurationIdFilter: []string{test.ConfigurationID(1)},
				},
			},
			want: func(t *testing.T, conditions map[string]store.Condition) {
				stored := make(map[string]store.Condition)
				for _, c := range conds {
					if c.Id == test.ConditionID(0) || c.ConfigurationId == test.ConfigurationID(1) {
						c.Versions = nil
						stored[c.Id] = c
					}
				}
				test.CmpStoredConditionMaps(t, stored, conditions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions := make(map[string]store.Condition)
			err := s.GetConditions(ctx, tt.args.owner, tt.args.query, func(c *store.Condition) error {
				condition, ok := conditions[c.Id]
				if ok {
					errM := test.MergeConditions(&condition, c)
					require.NoError(t, errM)
					conditions[c.Id] = condition
					return nil
				}
				conditions[c.Id] = *c.Clone()
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, conditions)
		})
	}
}
