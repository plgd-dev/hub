package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreUpdateAppliedConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	id := uuid.NewString()
	confID := uuid.NewString()
	condID := uuid.NewString()
	resources := []*pb.AppliedDeviceConfiguration_Resource{
		{
			Href:          "/test/1",
			CorrelationId: "corID",
		},
	}
	owner := "owner1"
	appliedConf, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Id:       id,
		DeviceId: "dev1",
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	})
	require.NoError(t, err)

	appliedConf2, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Id:       uuid.NewString(),
		DeviceId: "dev2",
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	})
	require.NoError(t, err)

	type args struct {
		update *pb.AppliedDeviceConfiguration
	}
	tests := []struct {
		name    string
		args    args
		want    func(*pb.AppliedDeviceConfiguration)
		wantErr bool
	}{
		{
			name: "missing Id",
			args: args{
				update: func() *pb.AppliedDeviceConfiguration {
					c := appliedConf.Clone()
					c.Id = ""
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "non-existing Id",
			args: args{
				update: func() *pb.AppliedDeviceConfiguration {
					c := appliedConf.Clone()
					c.Id = uuid.NewString()
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "non-matching owner",
			args: args{
				update: func() *pb.AppliedDeviceConfiguration {
					c := appliedConf.Clone()
					c.Owner = "owner2"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "duplicit deviceID+configurationID",
			args: args{
				update: func() *pb.AppliedDeviceConfiguration {
					c := appliedConf.Clone()
					c.Id = appliedConf2.GetId()
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				update: &pb.AppliedDeviceConfiguration{
					Id:       appliedConf.GetId(),
					Owner:    appliedConf.GetOwner(),
					DeviceId: "dev2",
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
						Id:      uuid.NewString(),
						Version: appliedConf.GetConfigurationId().GetVersion() + 1,
					},
					ExecutedBy: pb.MakeExecutedByOnDemand(),
					Resources: []*pb.AppliedDeviceConfiguration_Resource{
						{
							Href:          "/test/2",
							CorrelationId: "corID2",
							Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
						},
					},
				},
			},
			want: func(updated *pb.AppliedDeviceConfiguration) {
				require.Equal(t, appliedConf.GetId(), updated.GetId())
				require.Equal(t, "dev2", updated.GetDeviceId())
				require.Equal(t, appliedConf.GetOwner(), updated.GetOwner())
				require.Equal(t, pb.MakeExecutedByOnDemand(), updated.GetExecutedBy())
				require.Len(t, updated.GetResources(), 1)
				require.Equal(t, "/test/2", updated.GetResources()[0].GetHref())
				require.Equal(t, "corID2", updated.GetResources()[0].GetCorrelationId())
				require.Equal(t, pb.AppliedDeviceConfiguration_Resource_PENDING, updated.GetResources()[0].GetStatus())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := s.UpdateAppliedConfiguration(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(updated)
		})
	}
}

func TestStoreUpdateAppliedConfigurationPendingResources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	id := uuid.NewString()
	confID := uuid.NewString()
	condID := uuid.NewString()
	resources := []*pb.AppliedDeviceConfiguration_Resource{
		{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
		},
		{
			Href:          "/test/2",
			CorrelationId: "corID2",
			Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
			ResourceUpdated: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{DeviceId: "deviceID", Href: "/test/2"},
			},
		},
		{
			Href:          "/test/3",
			CorrelationId: "corID3",
			Status:        pb.AppliedDeviceConfiguration_Resource_QUEUED,
		},
	}
	owner := "owner1"
	_, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedDeviceConfiguration{
		Id:       id,
		DeviceId: "dev1",
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	})
	require.NoError(t, err)

	err = s.UpdateAppliedConfigurationPendingResource(ctx, owner, store.UpdateAppliedConfigurationPendingResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedDeviceConfiguration_Resource_DONE,
		},
	})
	require.NoError(t, err)

	// /test/1 is no longer in pending state, so additional update should fail
	err = s.UpdateAppliedConfigurationPendingResource(ctx, owner, store.UpdateAppliedConfigurationPendingResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedDeviceConfiguration_Resource_TIMEOUT,
		},
	})
	require.Error(t, err)

	// mismatched owner
	err = s.UpdateAppliedConfigurationPendingResource(ctx, "mismatch", store.UpdateAppliedConfigurationPendingResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedDeviceConfiguration_Resource{
			Href:          "/test/2",
			CorrelationId: "corID2",
			Status:        pb.AppliedDeviceConfiguration_Resource_DONE,
		},
	})
	require.Error(t, err)
}
