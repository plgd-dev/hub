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

func TestStoreDeleteConditions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

	getConditionsMap := func(t *testing.T, s *mongodb.Store, owner string, query *pb.GetConditionsRequest) map[string]store.Condition {
		conds := getConditions(t, s, owner, query)
		condsMap := make(map[string]store.Condition)
		for _, cond := range conds {
			condsMap[cond.Id] = *cond
		}
		return condsMap
	}

	type args struct {
		owner     string
		query     *pb.DeleteConditionsRequest
		makeQuery func(stored map[string]store.Condition) *pb.DeleteConditionsRequest
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
				condsMap := getConditionsMap(t, s, "", nil)
				require.NotEmpty(t, condsMap)
				newStored := make(map[string]store.Condition)
				for _, cond := range stored {
					if cond.Owner == test.Owner(1) {
						continue
					}
					newStored[cond.Id] = cond
				}
				test.CmpStoredConditionMaps(t, newStored, condsMap)
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
				condsMap := getConditionsMap(t, s, "", nil)
				require.NotEmpty(t, condsMap)
				newStored := make(map[string]store.Condition)
				for _, cond := range stored {
					if cond.Id == test.ConditionID(1) ||
						cond.Id == test.ConditionID(3) ||
						cond.Id == test.ConditionID(4) ||
						cond.Id == test.ConditionID(5) {
						continue
					}
					newStored[cond.Id] = cond
				}
				test.CmpStoredConditionMaps(t, newStored, condsMap)
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
				condsMap := getConditionsMap(t, s, "", nil)
				require.NotEmpty(t, condsMap)
				newStored := make(map[string]store.Condition)
				for _, cond := range stored {
					if cond.Id == test.ConditionID(2) {
						continue
					}
					newStored[cond.Id] = cond
				}
				test.CmpStoredConditionMaps(t, newStored, condsMap)
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
				for id, cond := range stored {
					cond.Versions = cond.Versions[:len(cond.Versions)-1]
					latest := cond.Versions[len(cond.Versions)-1].Copy()
					cond.Latest = &latest
					stored[id] = cond
				}
				confsMap := getConditionsMap(t, s, "", nil)
				require.NotEmpty(t, confsMap)
				test.CmpStoredConditionMaps(t, stored, confsMap)
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
				for id, cond := range stored {
					if cond.Owner != test.Owner(1) {
						continue
					}
					cond.Versions = cond.Versions[:len(cond.Versions)-1]
					latest := cond.Versions[len(cond.Versions)-1].Copy()
					cond.Latest = &latest
					stored[id] = cond
				}
				condsMap := getConditionsMap(t, s, "", nil)
				require.NotEmpty(t, condsMap)
				test.CmpStoredConditionMaps(t, stored, condsMap)
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
				condsMap := getConditionsMap(t, s, "", nil)
				test.CmpStoredConditionMaps(t, stored, condsMap)
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
				condsMap := getConditionsMap(t, s, "", nil)
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
				test.CmpStoredConditionMaps(t, stored, condsMap)
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
				condsMap := getConditionsMap(t, s, "", nil)
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
				test.CmpStoredConditionMaps(t, stored, condsMap)
			},
		},
		{
			name: "id7/version/{$(all)}",
			args: args{
				makeQuery: func(stored map[string]store.Condition) *pb.DeleteConditionsRequest {
					r := &pb.DeleteConditionsRequest{}
					id7, ok := stored[test.ConditionID(7)]
					require.True(t, ok)
					for _, version := range id7.Versions {
						r.IdFilter = append(r.IdFilter, &pb.IDFilter{
							Id:      test.ConditionID(7),
							Version: &pb.IDFilter_Value{Value: version.Version},
						})
					}
					return r
				},
			},
			want: func(t *testing.T, s *mongodb.Store, _ map[string]store.Condition) {
				conds := getConditionsMap(t, s, "", &pb.GetConditionsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(7),
						},
					},
				})
				require.Empty(t, conds)
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
			query := tt.args.query
			if tt.args.makeQuery != nil {
				query = tt.args.makeQuery(inserted)
			}
			_, err := s.DeleteConditions(ctx, tt.args.owner, query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, s, inserted)
		})
	}
}
