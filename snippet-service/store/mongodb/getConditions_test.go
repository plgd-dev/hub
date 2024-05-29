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
		want    func(t *testing.T, conditions []*store.Condition)
	}{
		{
			name: "all",
			args: args{
				owner: "",
				query: nil,
			},
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, len(conds))
			},
		},
		{
			name: "owner0",
			args: args{
				owner: test.Owner(0),
				query: nil,
			},
			want: func(t *testing.T, conditions []*store.Condition) {
				require.NotEmpty(t, conditions)
				for _, c := range conditions {
					require.Equal(t, test.Owner(0), c.Owner)
					cond, ok := conds[c.Id]
					require.True(t, ok)
					test.CmpStoredCondition(t, &cond, c, true)
				}
			},
		},
		{
			name: "id1/all",
			args: args{
				owner: "",
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, 1)
				c := conditions[0]
				cond, ok := conds[c.Id]
				require.True(t, ok)
				test.CmpStoredCondition(t, &cond, c, true)
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, 1)
				c := conditions[0]
				cond, ok := conds[c.Id]
				require.True(t, ok)
				require.Equal(t, test.ConditionID(2), cond.Id)
				require.Equal(t, test.Owner(2), cond.Owner)
				test.CmpStoredCondition(t, &cond, c, true)
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, test.RuntimeConfig.NumConditions)
				for _, c := range conditions {
					_, ok := conds[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 1)
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, 7)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(1), cond.Owner)
					require.Len(t, c.Versions, 1)
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Empty(t, conditions)
			},
		},
		{
			name: "owner2{latest, id2/latest, id5/latest} - non-matching owner", args: args{
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
			want: func(t *testing.T, conditions []*store.Condition) {
				require.Len(t, conditions, 6)
				for _, c := range conditions {
					cond, ok := conds[c.Id]
					require.True(t, ok)
					require.Equal(t, test.Owner(2), cond.Owner)
					require.Len(t, c.Versions, 1)
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
			want: func(t *testing.T, conditions []*store.Condition) {
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
			want: func(t *testing.T, conditions []*store.Condition) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conditions []*store.Condition
			err := s.GetConditions(ctx, tt.args.owner, tt.args.query, func(iterCtx context.Context, iter store.Iterator[store.Condition]) error {
				var cond store.Condition
				for iter.Next(iterCtx, &cond) {
					conditions = append(conditions, cond.Clone())
				}
				return iter.Err()
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
