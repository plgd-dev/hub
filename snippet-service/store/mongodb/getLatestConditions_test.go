package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreGetLatestConditions(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	const deviceID1 = "deviceID1"
	const deviceID2 = "deviceID2"
	const deviceID3 = "deviceID3"
	const href1 = "/1"
	const href2 = "/2"
	const href3 = "/3"
	const href4 = "/4"
	const href5 = "/5"
	const type1 = "type1"
	const type2 = "type2"
	const type3 = "type3"
	const owner1 = "owner1"
	const owner2 = "owner2"
	cond1In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c1",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		Owner:              owner1,
		JqExpressionFilter: ".test",
	}
	cond1, err := s.CreateCondition(ctx, cond1In)
	require.NoError(t, err)

	cond2In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c2",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID1},
		ResourceHrefFilter: []string{href1, href2, href3},
		ResourceTypeFilter: []string{type1, type2},
		Owner:              owner1,
	}
	cond2, err := s.CreateCondition(ctx, cond2In)
	require.NoError(t, err)

	cond3In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c3",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID2},
		ResourceHrefFilter: []string{href3, href4, href5},
		ResourceTypeFilter: []string{type3},
		Owner:              owner1,
	}
	cond3, err := s.CreateCondition(ctx, cond3In)
	require.NoError(t, err)

	cond4In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c4",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID1, deviceID3},
		ResourceHrefFilter: []string{href1, href5},
		ResourceTypeFilter: []string{type3},
		Owner:              owner1,
	}
	cond4, err := s.CreateCondition(ctx, cond4In)
	require.NoError(t, err)

	cond5In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c5",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID3},
		ResourceHrefFilter: []string{href1, href2},
		ResourceTypeFilter: []string{type2},
		Owner:              owner2,
	}
	cond5, err := s.CreateCondition(ctx, cond5In)
	require.NoError(t, err)

	cond6In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c6",
		Enabled:            true,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID2, deviceID3},
		ResourceHrefFilter: []string{href2, href3, href4},
		ResourceTypeFilter: []string{type1, type2, type3},
		Owner:              owner2,
	}
	cond6, err := s.CreateCondition(ctx, cond6In)
	require.NoError(t, err)

	cond7In := &pb.Condition{
		Id:              uuid.NewString(),
		Name:            "c7",
		Enabled:         true,
		ConfigurationId: uuid.NewString(),
		DeviceIdFilter:  []string{deviceID3},
		Owner:           owner2,
	}
	cond7, err := s.CreateCondition(ctx, cond7In)
	require.NoError(t, err)

	cond8In := &pb.Condition{
		Id:                 uuid.NewString(),
		Name:               "c8 - disabled",
		Enabled:            false,
		ConfigurationId:    uuid.NewString(),
		DeviceIdFilter:     []string{deviceID2, deviceID3},
		ResourceHrefFilter: []string{href2, href3, href4},
		ResourceTypeFilter: []string{type1, type2, type3},
		Owner:              owner2,
	}
	_, err = s.CreateCondition(ctx, cond8In)
	require.NoError(t, err)

	type args struct {
		query *store.GetLatestConditionsQuery
		owner string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Condition
	}{
		{
			name: "invalid query",
			args: args{
				query: &store.GetLatestConditionsQuery{},
			},
			wantErr: true,
		},
		{
			name: "deviceID3",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: deviceID3,
				},
			},
			want: []*pb.Condition{cond1, cond4, cond5, cond6, cond7},
		},
		{
			name: "owner1/deviceID1",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: deviceID1,
				},
				owner: owner1,
			},
			want: []*pb.Condition{cond1, cond2, cond4},
		},
		{
			name: "owner1/deviceID2",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: deviceID2,
				},
				owner: owner1,
			},
			want: []*pb.Condition{cond1, cond3},
		},
		{
			name: "owner1/new deviceID",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: uuid.NewString(),
				},
				owner: owner1,
			},
			want: []*pb.Condition{cond1},
		},
		{
			name: "owner2/deviceID1",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: deviceID1,
				},
				owner: owner2,
			},
			want: []*pb.Condition{},
		},
		{
			name: "owner2/deviceID3",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID: deviceID3,
				},
				owner: owner2,
			},
			want: []*pb.Condition{cond5, cond6, cond7},
		},
		{
			name: "href1",
			args: args{
				query: &store.GetLatestConditionsQuery{
					ResourceHref: href1,
				},
			},
			want: []*pb.Condition{cond1, cond2, cond4, cond5, cond7},
		},
		{
			name: "deviceID1/href3",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID:     deviceID1,
					ResourceHref: href3,
				},
			},
			want: []*pb.Condition{cond1, cond2},
		},
		{
			name: "owner2/href2",
			args: args{
				query: &store.GetLatestConditionsQuery{
					ResourceHref: href2,
				},
				owner: owner2,
			},
			want: []*pb.Condition{cond5, cond6, cond7},
		},
		{
			name: "[type2]",
			args: args{
				query: &store.GetLatestConditionsQuery{
					ResourceTypeFilter: []string{type2},
				},
			},
			want: []*pb.Condition{cond1, cond2, cond5, cond6, cond7},
		},
		{
			name: "deviceID2/[type3]",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID:           deviceID2,
					ResourceTypeFilter: []string{type3},
				},
			},
			want: []*pb.Condition{cond1, cond3, cond6},
		},
		{
			name: "owner2/[type1,type2,type3}",
			args: args{
				query: &store.GetLatestConditionsQuery{
					// order should not matter
					ResourceTypeFilter: []string{type2, type1, type3},
				},
				owner: owner2,
			},
			want: []*pb.Condition{cond6, cond7},
		},
		{
			name: "deviceID1/href5/[type3]",
			args: args{
				query: &store.GetLatestConditionsQuery{
					DeviceID:           deviceID1,
					ResourceHref:       href5,
					ResourceTypeFilter: []string{type3},
				},
			},
			want: []*pb.Condition{cond1, cond4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions := make(map[string]*pb.Condition)
			err := s.GetLatestEnabledConditions(ctx, tt.args.owner, tt.args.query, func(c *store.Condition) error {
				condition, errG := c.GetLatest()
				if errG != nil {
					return errG
				}
				conditions[c.Id] = condition.Clone()
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, conditions, len(tt.want))
			for _, c := range tt.want {
				latest, ok := conditions[c.GetId()]
				require.True(t, ok)
				test.CmpJSON(t, latest, c)
			}
		})
	}
}
