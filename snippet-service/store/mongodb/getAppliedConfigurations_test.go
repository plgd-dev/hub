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

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	appliedConfs := test.AddAppliedConfigurationsToStore(ctx, t, s)

	type args struct {
		owner string
		query *pb.GetAppliedConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(*store.AppliedConfiguration) bool
	}{
		{
			name: "all",
			args: args{},
			want: func(*store.AppliedConfiguration) bool {
				return true
			},
		},
		{
			name: "owner0",
			args: args{
				owner: test.Owner(0),
			},
			want: func(ac *store.AppliedConfiguration) bool {
				return ac.GetOwner() == test.Owner(0)
			},
		},
		{
			name: "id/{1, 3, 5}",
			args: args{
				query: &pb.GetAppliedConfigurationsRequest{
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
			want: func(ac *store.AppliedConfiguration) bool {
				acID := ac.GetId()
				return acID == test.AppliedConfigurationID(1) || acID == test.AppliedConfigurationID(3) || acID == test.AppliedConfigurationID(5)
			},
		},
		{
			name: "owner1/id{0, 1, 2, 3, 4, 5}",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetAppliedConfigurationsRequest{
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
			want: func(ac *store.AppliedConfiguration) bool {
				acID := ac.GetId()
				return (ac.GetOwner() == test.Owner(1)) &&
					(acID == test.AppliedConfigurationID(0) || acID == test.AppliedConfigurationID(1) || acID == test.AppliedConfigurationID(2) ||
						acID == test.AppliedConfigurationID(3) || acID == test.AppliedConfigurationID(4) || acID == test.AppliedConfigurationID(5))
			},
		},
		{
			name: "deviceId/{0, 2}",
			args: args{
				query: &pb.GetAppliedConfigurationsRequest{
					DeviceIdFilter: []string{
						test.DeviceID(0),
						test.DeviceID(2),
						// duplicates should be ignored
						test.DeviceID(2),
						test.DeviceID(0),
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acDeviceID := ac.GetDeviceId()
				return acDeviceID == test.DeviceID(0) || acDeviceID == test.DeviceID(2)
			},
		},
		{
			name: "owner2/{id{1, 2, 5} + deviceId{1, 3}}",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetAppliedConfigurationsRequest{
					IdFilter: []string{
						test.AppliedConfigurationID(1),
						test.AppliedConfigurationID(2),
						test.AppliedConfigurationID(5),
					},
					DeviceIdFilter: []string{
						test.DeviceID(1),
						test.DeviceID(3),
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acID := ac.GetId()
				acDeviceID := ac.GetDeviceId()
				return ac.GetOwner() == test.Owner(2) &&
					((acID == test.AppliedConfigurationID(1) || acID == test.AppliedConfigurationID(2) || acID == test.AppliedConfigurationID(5)) ||
						(acDeviceID == test.DeviceID(1) || acDeviceID == test.DeviceID(3)))
			},
		},
		{
			// get all owner0 configurations with a linkend configuration (should be all)
			name: "owner0/configurationId/all",
			args: args{
				owner: test.Owner(0),
				query: &pb.GetAppliedConfigurationsRequest{
					ConfigurationIdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				return ac.GetOwner() == test.Owner(0)
			},
		},
		{
			name: "owner1/configurationId/id{1, 3, 7}",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetAppliedConfigurationsRequest{
					ConfigurationIdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(1),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConfigurationID(3),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConfigurationID(7),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acConfID := ac.GetConfigurationId().GetId()
				return ac.GetOwner() == test.Owner(1) &&
					(acConfID == test.ConfigurationID(1) || acConfID == test.ConfigurationID(3) || acConfID == test.ConfigurationID(7))
			},
		},
		{
			name: "configurationId/version{1, 3, 7}",
			args: args{
				query: &pb.GetAppliedConfigurationsRequest{
					ConfigurationIdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 1,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 3,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 7,
							},
						},
					},
				},
			},

			want: func(ac *store.AppliedConfiguration) bool {
				acConfVersion := ac.GetConfigurationId().GetVersion()
				return acConfVersion == 1 || acConfVersion == 3 || acConfVersion == 7
			},
		},
		{
			name: "owner0/{deviceId{0, 2} + configurationId/version{1, 3}}",
			args: args{
				owner: test.Owner(0),
				query: &pb.GetAppliedConfigurationsRequest{
					DeviceIdFilter: []string{test.DeviceID(0), test.DeviceID(2)},
					ConfigurationIdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 1,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 3,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				if ac.GetOwner() != test.Owner(0) {
					return false
				}
				acConfVersion := ac.GetConfigurationId().GetVersion()
				acDeviceID := ac.GetDeviceId()
				return (acConfVersion == 1 || acConfVersion == 3) ||
					(acDeviceID == test.DeviceID(0) || acDeviceID == test.DeviceID(2))
			},
		},
		{
			// get all owner0 configurations with a linked condition (should be all)
			name: "owner1/conditionId/all",
			args: args{
				owner: test.Owner(1),
				query: &pb.GetAppliedConfigurationsRequest{
					ConditionIdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				return ac.GetOwner() == test.Owner(1)
			},
		},
		{
			name: "owner2/conditionId/id{0, 2, 4, 6}",
			args: args{
				owner: test.Owner(2),
				query: &pb.GetAppliedConfigurationsRequest{
					ConditionIdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(0),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConditionID(4),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConditionID(8),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConditionID(12),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acCondID := ac.GetConditionId().GetId()
				return ac.GetOwner() == test.Owner(2) &&
					(acCondID == test.ConditionID(0) || acCondID == test.ConditionID(4) || acCondID == test.ConditionID(8) || acCondID == test.ConditionID(12))
			},
		},
		{
			name: "owner0/{deviceId{0, 2} + conditionId/version{7, 9}}",
			args: args{
				owner: test.Owner(0),
				query: &pb.GetAppliedConfigurationsRequest{
					DeviceIdFilter: []string{test.DeviceID(0), test.DeviceID(2)},
					ConditionIdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 7,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 9,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acCondVersion := ac.GetConditionId().GetVersion()
				acDeviceID := ac.GetDeviceId()
				return (ac.GetOwner() == test.Owner(0)) && ((acCondVersion == 7 || acCondVersion == 9) ||
					(acDeviceID == test.DeviceID(0) || acDeviceID == test.DeviceID(2)))
			},
		},
		{
			name: "{id{0, 1} + deviceId{1, 2} + configurationId/id{3, 4} + conditionId/version{5, 6}}",
			args: args{
				query: &pb.GetAppliedConfigurationsRequest{
					IdFilter:       []string{test.AppliedConfigurationID(0), test.AppliedConfigurationID(1)},
					DeviceIdFilter: []string{test.DeviceID(1), test.DeviceID(2)},
					ConfigurationIdFilter: []*pb.IDFilter{
						{
							Id: test.ConfigurationID(3),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
						{
							Id: test.ConfigurationID(4),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
					ConditionIdFilter: []*pb.IDFilter{
						{
							Id: test.ConditionID(5),
							Version: &pb.IDFilter_Value{
								Value: 5,
							},
						},
						{
							Id: test.ConditionID(6),
							Version: &pb.IDFilter_Value{
								Value: 6,
							},
						},
					},
				},
			},
			want: func(ac *store.AppliedConfiguration) bool {
				acID := ac.GetId()
				acDeviceID := ac.GetDeviceId()
				acConfID := ac.GetConfigurationId().GetId()
				acCondID := ac.GetConditionId().GetId()
				acCondVersion := ac.GetConditionId().GetVersion()
				return (acID == test.AppliedConfigurationID(0) || acID == test.AppliedConfigurationID(1)) ||
					(acDeviceID == test.DeviceID(1) || acDeviceID == test.DeviceID(2)) ||
					(acConfID == test.ConfigurationID(3) || acConfID == test.ConfigurationID(4)) ||
					((acCondID == test.ConditionID(5) && acCondVersion == 5) ||
						(acCondID == test.ConditionID(6) && acCondVersion == 6))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appliedConfigurations := make(map[string]*pb.AppliedConfiguration)
			err := s.GetAppliedConfigurations(ctx, tt.args.owner, tt.args.query, func(c *store.AppliedConfiguration) error {
				appliedConfigurations[c.GetId()] = c.GetAppliedConfiguration().Clone()
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			stored := make(map[string]*pb.AppliedConfiguration)
			for _, ac := range appliedConfs {
				if tt.want(ac) {
					stored[ac.GetId()] = ac.GetAppliedConfiguration()
				}
			}
			require.Len(t, appliedConfigurations, len(stored))
			for _, c := range appliedConfigurations {
				ac, ok := stored[c.GetId()]
				require.True(t, ok)
				test.CmpJSON(t, ac, c)
			}
		})
	}
}

func TestGetExpiredAppliedConfigurationResourceUpdates(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	stored := test.AddAppliedConfigurationsToStore(ctx, t, s)

	got := make(map[string]*pb.AppliedConfiguration)
	validUntil, err := s.GetExpiredAppliedConfigurationResourceUpdates(ctx, func(ac *store.AppliedConfiguration) error {
		got[ac.GetId()] = ac.GetAppliedConfiguration().Clone()
		return nil
	})
	require.NoError(t, err)

	expiredStored := make(map[string]*pb.AppliedConfiguration)
	for _, ac := range stored {
		var resources []*pb.AppliedConfiguration_Resource
		for _, r := range ac.GetResources() {
			if r.GetStatus() == pb.AppliedConfiguration_Resource_PENDING &&
				r.GetValidUntil() > 0 && r.GetValidUntil() <= validUntil {
				resources = append(resources, r)
			}
		}
		if len(resources) > 0 {
			newAc := ac.Clone()
			newAc.Resources = resources
			expiredStored[ac.GetId()] = newAc
		}
	}

	require.Len(t, got, len(expiredStored))
	test.CmpAppliedDeviceConfigurationsMaps(t, expiredStored, got, false)
}
