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

func TestStoreDeleteConditions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	getConditions := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetConditionsRequest) []*store.Condition {
		var conditions []*store.Condition
		err := s.GetConditions(ctx, owner, query, func(iterCtx context.Context, iter store.Iterator[store.Condition]) error {
			var cond store.Condition
			for iter.Next(iterCtx, &cond) {
				conditions = append(conditions, cond.Clone())
			}
			return iter.Err()
		})
		require.NoError(t, err)
		return conditions
	}

	type args struct {
		owner string
		query *pb.DeleteConditionsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition)
	}{
		{
			name: "all",
			args: args{},
			want: func(t *testing.T, s *mongodb.Store, _ map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				require.Empty(t, conds)
			},
		},
		{
			name: "owner1",
			args: args{
				owner: test.Owner(1),
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				require.NotEmpty(t, conds)
				newCount := 0
				for _, cond := range conds {
					require.NotEqual(t, test.Owner(1), cond.Owner)
					newCount += len(cond.Versions)
				}
				storedCount := 0
				for _, cond := range stored {
					if cond.Owner != test.Owner(1) {
						storedCount += len(cond.Versions)
					}
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "id{1,3,4,5}",
			args: args{
				query: &pb.DeleteConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id:      test.ConditionID(1),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConditionID(3),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConditionID(4),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConditionID(5),
							Version: &pb.IDFilter_All{All: true},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				require.NotEmpty(t, conds)
				newCount := 0
				for _, cond := range conds {
					require.NotEqual(t, test.ConditionID(1), cond.Id)
					require.NotEqual(t, test.ConditionID(3), cond.Id)
					require.NotEqual(t, test.ConditionID(4), cond.Id)
					require.NotEqual(t, test.ConditionID(5), cond.Id)
					newCount += len(cond.Versions)
				}
				storedCount := 0
				for _, cond := range stored {
					if cond.Id == test.ConditionID(1) ||
						cond.Id == test.ConditionID(3) ||
						cond.Id == test.ConditionID(4) ||
						cond.Id == test.ConditionID(5) {
						continue
					}
					storedCount += len(cond.Versions)
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "owner2/id2",
			args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id:      test.ConditionID(2),
							Version: &pb.IDFilter_All{All: true},
						},
						// Ids not owned by owner2 should not be deleted
						{
							Id:      test.ConditionID(1),
							Version: &pb.IDFilter_All{All: true},
						},
						{
							Id:      test.ConditionID(3),
							Version: &pb.IDFilter_All{All: true},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				require.NotEmpty(t, conds)
				newCount := 0
				for _, cond := range conds {
					require.NotEqual(t, test.ConditionID(2), cond.Id)
					newCount += len(cond.Versions)
				}
				storedCount := 0
				for _, cond := range stored {
					if cond.Id == test.ConditionID(2) {
						continue
					}
					storedCount += len(cond.Versions)
				}
				require.Equal(t, storedCount, newCount)
			},
		},
		{
			name: "latest",
			args: args{
				query: &pb.DeleteConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				storedLatest := make(map[string]store.ConditionVersion)
				storedCount := 0
				for _, cond := range stored {
					storedLatest[cond.Id] = cond.Versions[len(cond.Versions)-1]
					storedCount += len(cond.Versions)
				}
				conds := getConditions(t, s, "", nil)
				require.NotEmpty(t, conds)
				count := 0
				for _, cond := range conds {
					require.NotEqual(t, storedLatest[cond.Id], cond.Versions[len(cond.Versions)-1])
					count += len(cond.Versions)
				}
				require.Equal(t, storedCount-len(storedLatest), count)
			},
		},
		{
			name: "owner1/latest",
			args: args{
				owner: test.Owner(1),
				query: &pb.DeleteConditionsRequest{
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
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				storedLatest := make(map[string]store.ConditionVersion)
				storedCount := 0
				for _, cond := range stored {
					storedLatest[cond.Id] = cond.Versions[len(cond.Versions)-1]
					storedCount += len(cond.Versions)
				}
				conds := getConditions(t, s, "", nil)
				require.NotEmpty(t, conds)
				count := 0
				removed := 0
				for _, cond := range conds {
					if cond.Owner == test.Owner(1) {
						require.NotEqual(t, storedLatest[cond.Id], cond.Versions[len(cond.Versions)-1])
						removed++
					} else {
						require.Equal(t, storedLatest[cond.Id], cond.Versions[len(cond.Versions)-1])
					}
					count += len(cond.Versions)
				}
				require.Equal(t, storedCount-removed, count)
			},
		},
		{
			name: "owner2/id1/latest - non-matching owner",
			args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConditionsRequest{
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
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				condsMap := make(map[string]store.Condition)
				for _, cond := range conds {
					condsMap[cond.Id] = *cond
				}
				test.CmpStoredConditionsMaps(t, stored, condsMap)
			},
		},
		{
			name: "version/{13, 113, 213, 313, 413, 513}", args: args{
				owner: "",
				query: &pb.DeleteConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{Version: &pb.IDFilter_Value{Value: 13}},
						{Version: &pb.IDFilter_Value{Value: 113}},
						{Version: &pb.IDFilter_Value{Value: 213}},
						{Version: &pb.IDFilter_Value{Value: 313}},
						{Version: &pb.IDFilter_Value{Value: 413}},
						{Version: &pb.IDFilter_Value{Value: 513}},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				condsMap := make(map[string]store.Condition)
				for _, cond := range conds {
					condsMap[cond.Id] = *cond
				}
				for _, cond := range stored {
					versions := make([]store.ConditionVersion, 0)
					for _, version := range cond.Versions {
						if version.Version == 13 ||
							version.Version == 113 ||
							version.Version == 213 ||
							version.Version == 313 ||
							version.Version == 413 ||
							version.Version == 513 {
							continue
						}
						versions = append(versions, version)
					}
					cond.Versions = versions
					stored[cond.Id] = cond
				}
				test.CmpStoredConditionsMaps(t, stored, condsMap)
			},
		},
		{
			name: "owner2/version/{207, 213, 221}", args: args{
				owner: test.Owner(2),
				query: &pb.DeleteConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{Version: &pb.IDFilter_Value{Value: 207}},
						{Version: &pb.IDFilter_Value{Value: 213}},
						{Version: &pb.IDFilter_Value{Value: 221}},
						// duplicates should be ignored
						{Version: &pb.IDFilter_Value{Value: 213}},
						// filter with Id should be ignored if there are filters without Id
						{
							Id:      test.ConditionID(2),
							Version: &pb.IDFilter_Value{Value: 213},
						},
					},
				},
			},
			want: func(t *testing.T, s *mongodb.Store, stored map[string]store.Condition) {
				conds := getConditions(t, s, "", nil)
				condsMap := make(map[string]store.Condition)
				for _, cond := range conds {
					condsMap[cond.Id] = *cond
				}
				for _, cond := range stored {
					if cond.Owner == test.Owner(2) {
						versions := make([]store.ConditionVersion, 0)
						for _, version := range cond.Versions {
							if version.Version == 207 ||
								version.Version == 213 ||
								version.Version == 221 {
								continue
							}
							versions = append(versions, version)
						}
						cond.Versions = versions
						stored[cond.Id] = cond
					}
				}
				test.CmpStoredConditionsMaps(t, stored, condsMap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, cleanUpStore := test.NewMongoStore(t)
			defer cleanUpStore()
			inserted := test.AddConditionsToStore(ctx, t, s, 500, func(iteration int) uint64 {
				return uint64(iteration * 100)
			})
			_, err := s.DeleteConditions(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, s, inserted)
		})
	}
}
