package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreGetAppliedConfigurations(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()
	appliedConfs := test.AddAppliedConfigurationsToStore(ctx, t, s)

	type args struct {
		owner string
		query *pb.GetAppliedDeviceConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(confs map[string]*store.AppliedDeviceConfiguration)
	}{
		{
			name: "all",
			args: args{},
			want: func(confs map[string]*store.AppliedDeviceConfiguration) {
				require.Len(t, confs, len(appliedConfs))
				for _, c := range confs {
					_, ok := appliedConfs[c.GetId()]
					require.True(t, ok)
					test.CmpJSON(t, appliedConfs[c.GetId()], c)
				}
			},
		},
		{
			name: "owner0",
			args: args{
				owner: test.Owner(0),
			},
			want: func(confs map[string]*store.AppliedDeviceConfiguration) {
				require.NotEmpty(t, confs)
				for _, c := range confs {
					require.Equal(t, test.Owner(0), c.GetOwner())
					appliedConf, ok := appliedConfs[c.GetId()]
					require.True(t, ok)
					test.CmpJSON(t, appliedConf, c)
				}
			},
		},
		{
			name: "id/{1, 3, 5}",
			args: args{
				query: &pb.GetAppliedDeviceConfigurationsRequest{
					IdFilter: []string{
						test.AppliedConfigurationID(1),
						test.AppliedConfigurationID(3),
						test.AppliedConfigurationID(5),
						// duplicates should be ignored
						test.AppliedConfigurationID(5),
						test.AppliedConfigurationID(5),
					},
				},
			},
			want: func(confs map[string]*store.AppliedDeviceConfiguration) {
				stored := make(map[string]*store.AppliedDeviceConfiguration)
				for _, ac := range appliedConfs {
					if ac.GetId() == test.AppliedConfigurationID(1) ||
						ac.GetId() == test.AppliedConfigurationID(3) ||
						ac.GetId() == test.AppliedConfigurationID(5) {
						stored[ac.GetId()] = ac
					}
				}
				require.Len(t, confs, len(stored))
				for _, c := range confs {
					ac, ok := stored[c.GetId()]
					require.True(t, ok)
					test.CmpJSON(t, ac, c)
				}
			},
		},
		{
			name: "owner2/id{0, 1, 2, 3, 4, 5}",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetAppliedDeviceConfigurationsRequest{
					IdFilter: []string{
						test.AppliedConfigurationID(0),
						test.AppliedConfigurationID(1),
						test.AppliedConfigurationID(2),
						test.AppliedConfigurationID(3),
						test.AppliedConfigurationID(4),
						test.AppliedConfigurationID(5),
					},
				},
			},
			want: func(confs map[string]*store.AppliedDeviceConfiguration) {
				stored := make(map[string]*store.AppliedDeviceConfiguration)
				for _, ac := range appliedConfs {
					if ac.GetOwner() != test.Owner(2) {
						continue
					}
					acID := ac.GetId()
					if acID == test.AppliedConfigurationID(0) ||
						acID == test.AppliedConfigurationID(1) ||
						acID == test.AppliedConfigurationID(2) ||
						acID == test.AppliedConfigurationID(3) ||
						acID == test.AppliedConfigurationID(4) ||
						acID == test.AppliedConfigurationID(5) {
						stored[acID] = ac
					}
				}
				require.Len(t, confs, len(stored))
				for _, c := range confs {
					ac, ok := stored[c.GetId()]
					require.True(t, ok)
					test.CmpJSON(t, ac, c)
				}
			},
		},
		{
			name: "configurationId/{2, 4, 7}",
			args: args{
				query: &pb.GetAppliedDeviceConfigurationsRequest{
					ConfigurationIdFilter: []string{
						test.ConfigurationID(2),
						test.ConfigurationID(4),
						test.ConfigurationID(7),
						// duplicates should be ignored
						test.ConfigurationID(7),
						test.ConfigurationID(4),
						test.ConfigurationID(2),
					},
				},
			},
			want: func(confs map[string]*store.AppliedDeviceConfiguration) {
				stored := make(map[string]*store.AppliedDeviceConfiguration)
				for _, ac := range appliedConfs {
					acConfId := ac.GetConfigurationId().GetId()
					if acConfId == test.ConfigurationID(2) || acConfId == test.ConfigurationID(4) || acConfId == test.ConfigurationID(7) {
						stored[ac.GetId()] = ac
					}
				}
				require.Len(t, confs, len(stored))
				for _, c := range confs {
					ac, ok := stored[c.GetId()]
					require.True(t, ok)
					test.CmpJSON(t, ac, c)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appliedConfigurations := make(map[string]*store.AppliedDeviceConfiguration)
			err := s.GetAppliedDeviceConfigurations(ctx, tt.args.owner, tt.args.query, func(c *store.AppliedDeviceConfiguration) error {
				appliedConfigurations[c.GetId()] = c.Clone()
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(appliedConfigurations)
		})
	}
}
